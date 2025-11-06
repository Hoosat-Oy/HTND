package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/client"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/daemon/pb"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/keys"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/libhtnwallet"
	"github.com/Hoosat-Oy/HTND/cmd/htnwallet/utils"
	"github.com/pkg/errors"
)

func autoCompound(conf *autoCompoundConfig) error {
	fmt.Println("Hoosat Auto-Compounder STARTED → 1 compound tx every 10 seconds")

	// === Load keys ===
	keysFile, err := keys.ReadKeysFile(conf.NetParams(), conf.KeysFile)
	if err != nil {
		return errors.Wrap(err, "reading keys file")
	}

	if len(keysFile.ExtendedPublicKeys) > len(keysFile.EncryptedMnemonics) {
		return errors.New("multisig wallet detected but not all private keys present")
	}

	if len(conf.Password) == 0 {
		conf.Password = keys.GetPassword("Enter wallet password: ")
	}

	mnemonics, err := keysFile.DecryptMnemonics(conf.Password)
	if err != nil {
		return errors.Wrap(err, "wrong password")
	}

	// === Connect to htnwallet daemon ===
	daemonClient, tearDown, err := client.Connect(conf.DaemonAddress)
	if err != nil {
		return errors.Wrap(err, "connecting to htnwallet daemon")
	}
	defer tearDown()

	// === Amount ===
	var sendAmountSompi uint64
	if !conf.IsSendAll {
		sendAmountSompi, err = utils.KasToSompi(conf.SendAmount)
		if sendAmountSompi < 16*88 {
			sendAmountSompi = 16 * 88
		}
		if err != nil {
			sendAmountSompi = 16 * 88
		}
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		if err := compoundOnce(conf, daemonClient, mnemonics, keysFile.ECDSA, sendAmountSompi); err != nil {
			fmt.Printf("[%s] compound failed: %v\n", time.Now().Format("15:04:05"), err)
			continue
		}
	}
}

func compoundOnce(
	conf *autoCompoundConfig,
	client pb.HtnwalletdClient, // CORRECT TYPE
	mnemonics []string,
	ecdsa bool,
	amount uint64,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), daemonTimeout)
	defer cancel()

	// 1. Create unsigned tx
	resp, err := client.CreateUnsignedTransactions(ctx, &pb.CreateUnsignedTransactionsRequest{
		From:                     conf.FromAddresses,
		Address:                  conf.ToAddress,
		Amount:                   amount,
		IsSendAll:                conf.IsSendAll,
		UseExistingChangeAddress: conf.UseExistingChangeAddress,
	})
	if err != nil {
		fmt.Printf("[%s] NOTHING TO COMPOUND → Error: %s\n",
			time.Now().Format("15:04:05"), err)
		return nil
	}

	if len(resp.UnsignedTransactions) == 0 {
		fmt.Printf("[%s] NOTHING TO COMPOUND\n",
			time.Now().Format("15:04:05"))
		return nil
	}

	unsignedTx := resp.UnsignedTransactions[0]

	// 2. Sign
	signedTx, err := libhtnwallet.Sign(conf.NetParams(), mnemonics, unsignedTx, ecdsa)
	if err != nil {
		return errors.Wrap(err, "signing failed")
	}

	// 3. Broadcast
	bctx, bcancel := context.WithTimeout(context.Background(), daemonTimeout)
	defer bcancel()

	bresp, err := client.Broadcast(bctx, &pb.BroadcastRequest{
		Transactions: [][]byte{signedTx},
	})
	if err != nil {
		return errors.Wrap(err, "broadcast failed")
	}

	// 4. Success
	for _, txid := range bresp.TxIDs {
		fmt.Printf("[%s] COMPOUNDED → https://explorer.hoosat.fi/txs/%s\n",
			time.Now().Format("15:04:05"), txid)
	}

	return nil
}
