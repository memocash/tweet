package bot

import (
	"fmt"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/syndtr/goleveldb/leveldb"
	"net/http"
)

type InfoServer struct {
	Db *leveldb.DB
}

func NewInfoServer(db *leveldb.DB) *InfoServer {
	return &InfoServer{
		Db: db,
	}
}

func (l *InfoServer) Listen() error {
	cfg := config.GetConfig()
	if cfg.InfoServerPort == 0 {
		return fmt.Errorf("port is 0 for info server")
	}
	mux := http.NewServeMux()
	db := database.Database{Db: l.Db}
	mux.HandleFunc("/balance", func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			writer.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
			return
		}
		address := request.FormValue("address")
		addr, err := wallet.GetAddrFromString(address)
		if err != nil {
			writer.Write([]byte(fmt.Sprintf("error getting address; %v", err)))
			return
		}
		utxos, err := db.GetUtxos(*addr)
		var total int64
		for _, utxo := range utxos {
			writer.Write([]byte(fmt.Sprintf("utxo: %s:%d - %d\n", utxo.Hash, utxo.Index, utxo.Amount)))
			total += utxo.Amount
		}
		writer.Write([]byte(fmt.Sprintf("balance: %d", total)))
	})
	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.InfoServerPort), mux)
	return fmt.Errorf("error listening for info api: %w", err)
}
