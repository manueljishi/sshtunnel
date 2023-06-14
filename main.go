package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/gliderlabs/ssh"
)

type SshFile struct {
	fileContents []byte
	isDone       bool
	id           int
	doneCh       chan bool
}

var filesMap = make(map[int]SshFile)

func main() {

	go func() {
		http.HandleFunc("/", handleRequest)
		log.Fatal(http.ListenAndServe(":3000", nil))
	}()

	ssh.Handle(func(s ssh.Session) {
		sessionId := rand.Intn(math.MaxInt32)
		s.Write([]byte(fmt.Sprintf("Session id is %d\n", sessionId)))
		file := SshFile{
			isDone: false,
			id:     sessionId,
			doneCh: make(chan bool, 1),
		}
		content, err := ioutil.ReadAll(s)
		if err != nil {
			// Handle the error
			// For example, you can log the error or return an error message to the client
			s.Write([]byte(err.Error()))
		}
		file.fileContents = content
		filesMap[sessionId] = file
		file.isDone = true
		filesMap[sessionId].doneCh <- true
		close(file.doneCh)
		s.Write([]byte("Done serving file\n"))

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
	file, ok := filesMap[sessionId]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		notFoundFile, err := os.ReadFile("./404.html")
		if err != nil {
			log.Println(err)
		}
		w.Write(notFoundFile)
		return
	}
	if !file.isDone {
		<-file.doneCh
	}
	w.WriteHeader(http.StatusOK)
	w.Write(file.fileContents)
	fmt.Println("Served file")
}
