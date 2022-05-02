// Converts magnet URIs and info hashes into torrent metainfo files.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/anacrolix/envpprof"
	"github.com/anacrolix/tagflag"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/cute-angelia/go-utils/syntax/ijson"
)

func main() {
	args := struct {
		tagflag.StartPos
		Magnet []string
	}{}
	tagflag.Parse(&args)
	cl, err := torrent.NewClient(nil)
	if err != nil {
		log.Fatalf("error creating client: %s", err)
	}
	http.HandleFunc("/torrent", func(w http.ResponseWriter, r *http.Request) {
		cl.WriteStatus(w)
	})
	http.HandleFunc("/dht", func(w http.ResponseWriter, r *http.Request) {
		for _, ds := range cl.DhtServers() {
			ds.WriteStatus(w)
		}
	})
	wg := sync.WaitGroup{}
	for _, arg := range args.Magnet {
		t, err := cl.AddMagnet(arg)
		if err != nil {
			log.Fatalf("error adding magnet to client: %s", err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			startTime := time.Now()

			<-t.GotInfo()
			mi := t.Metainfo()

			// 修改后赋值
			nName := ""
			for _, runeValue := range t.Info().Name {
				nName += fmt.Sprintf("_%c", runeValue)
			}
			t.Info().Name = nName
			mi.InfoBytes, _ = bencode.Marshal(t.Info())
			log.Println(ijson.Pretty(t.Info()))

			t.Drop()
			f, err := os.Create(t.Info().Name + ".torrent")
			if err != nil {
				log.Fatalf("error creating torrent metainfo file: %s", err)
			}
			defer f.Close()

			log.Println("转化种子成功", time.Since(startTime))
			err = bencode.NewEncoder(f).Encode(mi)
			if err != nil {
				log.Fatalf("error writing torrent metainfo file: %s", err)
			}
		}()
	}
	wg.Wait()
}
