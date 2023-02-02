package bot

import (
	"encoding/json"
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
	mux.HandleFunc("/profile", func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			writer.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
			return
		}
		sender := request.FormValue("sender")
		twittername := request.FormValue("twittername")
		profileBytes, err := db.Db.Get([]byte(fmt.Sprintf("profile-%s-%s", sender, twittername)), nil)
		writer.Write([]byte(fmt.Sprintf("Searching for profile-%s-%s\n", sender, twittername)))
		if err != nil {
			writer.Write([]byte(fmt.Sprintf("error getting profile; %v", err)))
			return
		}
		var profile database.Profile
		json.Unmarshal(profileBytes, &profile)
		writer.Write([]byte(fmt.Sprintf("name: %v\ndesc: %v\npicUrl: %v\n", profile.Name, profile.Description, profile.ProfilePic)))
	})

	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.InfoServerPort), mux)
	return fmt.Errorf("error listening for info api: %w", err)
}
