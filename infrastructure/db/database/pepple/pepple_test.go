package pepple

import (
	"os"
	"reflect"
	"testing"

	"github.com/Hoosat-Oy/HTND/infrastructure/db/database"
)

func prepareDatabaseForTest(t *testing.T, testName string) (ldb *PeppleDB, teardownFunc func()) {
	// Create a temp db to run tests against
	path, err := os.MkdirTemp("", testName)
	if err != nil {
		t.Fatalf("%s: TempDir unexpectedly failed: %s", testName, err)
	}
	ldb, err = NewPeppleDB(path, 8)
	if err != nil {
		t.Fatalf("%s: NewPeppleDB unexpectedly failed: %s", testName, err)
	}
	teardownFunc = func() {
		err = ldb.Close()
		if err != nil {
			t.Fatalf("%s: Close unexpectedly failed: %s", testName, err)
		}
	}
	return ldb, teardownFunc
}

func TestPeppleDBSanity(t *testing.T) {
	ldb, teardownFunc := prepareDatabaseForTest(t, "TestPeppleDBSanity")
	defer teardownFunc()

	// Put something into the db
	key := database.MakeBucket(nil).Key([]byte("key"))
	putData := []byte("Hello world!")
	err := ldb.Put(key, putData)
	if err != nil {
		t.Fatalf("TestPeppleDBSanity: Put returned unexpected error: %s", err)
	}

	// Get from the key previously put to
	getData, err := ldb.Get(key)
	if err != nil {
		t.Fatalf("TestPeppleDBSanity: Get returned unexpected error: %s", err)
	}

	// Make sure that the put data and the get data are equal
	if !reflect.DeepEqual(getData, putData) {
		t.Fatalf("TestPeppleDBSanity: get data and put data are not equal. Put: %s, got: %s",
			string(putData), string(getData))
	}
}

func TestPeppleDBTransactionSanity(t *testing.T) {
	ldb, teardownFunc := prepareDatabaseForTest(t, "TestPeppleDBTransactionSanity")
	defer teardownFunc()

	// Case 1. Write in tx and then read directly from the DB
	// Begin a new transaction
	tx, err := ldb.Begin()
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Begin unexpectedly failed: %s", err)
	}

	// Put something into the transaction
	key := database.MakeBucket(nil).Key([]byte("key"))
	putData := []byte("Hello world!")
	err = tx.Put(key, putData)
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Put returned unexpected error: %s", err)
	}

	// Get from the key previously put to. Since the tx is not yet committed, this should return ErrNotFound.
	_, err = ldb.Get(key)
	if err == nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Get unexpectedly succeeded")
	}
	if !database.IsNotFoundError(err) {
		t.Fatalf("TestPeppleDBTransactionSanity: Get returned wrong error: %s", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Commit returned unexpected error: %s", err)
	}

	// Get from the key previously put to. Now that the tx was committed, this should succeed.
	getData, err := ldb.Get(key)
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Get returned unexpected error: %s", err)
	}

	// Make sure that the put data and the get data are equal
	if !reflect.DeepEqual(getData, putData) {
		t.Fatalf("TestPeppleDBTransactionSanity: get data and put data are not equal. Put: %s, got: %s",
			string(putData), string(getData))
	}

	// Case 2. Write directly to the DB and then read from a tx
	// Put something into the db
	key = database.MakeBucket(nil).Key([]byte("key2"))
	putData = []byte("Goodbye world!")
	err = ldb.Put(key, putData)
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Put returned unexpected error: %s", err)
	}

	// Begin a new transaction
	tx, err = ldb.Begin()
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Begin unexpectedly failed: %s", err)
	}

	// Get from the key previously put to
	getData, err = tx.Get(key)
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: Get returned unexpected error: %s", err)
	}

	// Make sure that the put data and the get data are equal
	if !reflect.DeepEqual(getData, putData) {
		t.Fatalf("Test MioTestPeppleDBTransactionSanity: get data and put data are not equal. Put: %s, got: %s",
			string(putData), string(getData))
	}

	// Rollback the transaction
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("TestPeppleDBTransactionSanity: rollback returned unexpected error: %s", err)
	}
}
