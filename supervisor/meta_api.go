package supervisor

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/crypto/ssh"
)

type MetaAPI struct {
	PrivateKeyFile string
}

func (self MetaAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/meta/pubkey`):
		privateBytes, err := ioutil.ReadFile(self.PrivateKeyFile)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		private, err := ssh.ParsePrivateKey(privateBytes)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		public := private.PublicKey()
		w.Write([]byte(
			fmt.Sprintf("%s %s\n",
				public.Type(),
				base64.StdEncoding.EncodeToString(public.Marshal()))))
		return
	}

	w.WriteHeader(501)
	return
}
