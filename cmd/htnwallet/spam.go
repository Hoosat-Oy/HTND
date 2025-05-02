package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/client"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/pb"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/keys"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/libhtnwallet"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/utils"
	"github.com/pkg/errors"
)

func spamSend(conf *spamConfig) error {
	fmt.Println("Trying to spam, make sure testnet!")

	if !conf.NetworkFlags.Testnet {
		return errors.New("spamSend is intended for testnet use only")
	}

	tps, err := strconv.ParseUint(conf.TxsPerSecond, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse TPS: %w", err)
	}

	fmt.Printf("Starting spam send at %d TPS. Press Ctrl+C to stop.\n", tps)
	ticker := time.NewTicker(time.Second / time.Duration(tps))
	defer ticker.Stop()

	// Load keys
	keysFile, err := keys.ReadKeysFile(conf.NetParams(), conf.KeysFile)
	if err != nil {
		return err
	}

	if len(keysFile.ExtendedPublicKeys) > len(keysFile.EncryptedMnemonics) {
		return errors.New("cannot use 'send' command for multisig wallet without all of the keys")
	}

	// Connect to daemon
	daemonClient, tearDown, err := client.Connect(conf.DaemonAddress)
	if err != nil {
		return err
	}
	defer tearDown()

	var sendAmountSompi uint64
	if !conf.IsSendAll {
		sendAmountSompi, err = utils.KasToSompi(conf.SendAmount)
		if err != nil {
			return err
		}
	}

	// Prompt for password if missing
	if conf.Password == "" {
		conf.Password = keys.GetPassword("Password:")
	}

	mnemonics, err := keysFile.DecryptMnemonics(conf.Password)
	if err != nil {
		if strings.Contains(err.Error(), "message authentication failed") {
			fmt.Fprintln(os.Stderr, "Password decryption failed. This can happen if the wrong keys file is used.")
		}
		return err
	}

	// Start spam loop
	for range ticker.C {
		go func() {
			failure := false
			for attempt := 1; attempt <= maxRetries; attempt++ {
				ctx, cancel := context.WithTimeout(context.Background(), daemonTimeout)
				defer cancel()

				// Create unsigned transactions
				req := &pb.CreateUnsignedTransactionsRequest{
					From:                     conf.FromAddresses,
					Address:                  conf.ToAddress,
					Amount:                   sendAmountSompi,
					IsSendAll:                conf.IsSendAll,
					UseExistingChangeAddress: conf.UseExistingChangeAddress,
				}

				unsignedResp, err := daemonClient.CreateUnsignedTransactions(ctx, req)
				if err != nil {
					continue
				}

				signedTxs := make([][]byte, len(unsignedResp.UnsignedTransactions))
				for i, tx := range unsignedResp.UnsignedTransactions {
					signedTx, err := libhtnwallet.Sign(conf.NetParams(), mnemonics, tx, keysFile.ECDSA)
					if err != nil {
						failure = true
						break
					}
					signedTxs[i] = signedTx
				}

				// Broadcast transactions in chunks
				if !failure {
					chunkSize := 100
					for offset := 0; offset < len(signedTxs); offset += chunkSize {
						end := offset + chunkSize
						if end > len(signedTxs) {
							end = len(signedTxs)
						}
						chunk := signedTxs[offset:end]

						broadcastCtx, broadcastCancel := context.WithTimeout(context.Background(), daemonTimeout)
						resp, err := daemonClient.Broadcast(broadcastCtx, &pb.BroadcastRequest{Transactions: chunk})
						broadcastCancel()
						if err != nil {
							failure = true
							break
						}

						fmt.Printf("Broadcasted %d transaction(s) (%0.2f%% complete)\n", len(chunk), 100*float64(end)/float64(len(signedTxs)))
						for _, txID := range resp.TxIDs {
							fmt.Printf("\t%s\n", txID)
						}
					}
				}
				if !failure {
					if conf.Verbose {
						fmt.Println("Serialized Transactions:")
						for _, tx := range signedTxs {
							fmt.Printf("\t%x\n", tx)
						}
					}
				} else {
					if attempt < maxRetries {
						time.Sleep(retryDelay)
						continue
					} else {
						break
					}
				}
			}
		}()
	}

	return nil
}
