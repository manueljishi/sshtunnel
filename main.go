package main

import (
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/gliderlabs/ssh"
)

type SshFile struct {
	w      io.Writer
	donech chan struct{}
	id     int
}

var files = map[int]chan SshFile{}

func main() {

	go func() {
		http.HandleFunc("/", handleRequest)
		log.Fatal(http.ListenAndServe(":3000", nil))
	}()

	ssh.Handle(func(s ssh.Session) {
		sessionId := rand.Intn(math.MaxInt32)
		files[sessionId] = make(chan SshFile)
		log.Printf("Session id is %d\n", sessionId)
		file := <-files[sessionId]
		file.id = sessionId
		_, err := io.Copy(file.w, s)
		if err != nil {
			log.Printf("Failed to copy data, %s\n", err)
		}
		close(file.donech)

	})

	log.Fatal(ssh.ListenAndServe(":2222", nil, ssh.HostKeyFile("/var/ssh-server/id_rsa")))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	sessionId, err := strconv.Atoi(id)
	if len(id) == 0 || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing id parameter"))
		return
	}
	fileChan, ok := files[sessionId]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		notFoundFile, err := os.ReadFile("./404.html")
		if err != nil {
			log.Println(err)
		}
		w.Write(notFoundFile)
		return
	}
	donech := make(chan struct{})
	fileChan <- SshFile{
		w:      w,
		donech: donech,
	}
	<-donech
}
