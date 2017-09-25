package micro

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	boshdispatcher "github.com/cloudfoundry/bosh-agent/httpsdispatcher"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	"github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type HTTPSHandler struct {
	parsedURL   *url.URL
	logger      boshlog.Logger
	dispatcher  *boshdispatcher.HTTPSDispatcher
	fs          boshsys.FileSystem
	dirProvider boshdir.Provider
}

func NewHTTPSHandler(
	parsedURL *url.URL,
	logger boshlog.Logger,
	fs boshsys.FileSystem,
	dirProvider boshdir.Provider,
) (handler HTTPSHandler) {
	handler.parsedURL = parsedURL
	handler.logger = logger
	handler.fs = fs
	handler.dirProvider = dirProvider
	handler.dispatcher = boshdispatcher.NewHTTPSDispatcher(parsedURL, logger)
	return
}

func (h HTTPSHandler) Run(handlerFunc boshhandler.Func) error {
	err := h.Start(handlerFunc)
	if err != nil {
		return bosherr.WrapError(err, "Starting https handler")
	}
	return nil
}

func (h HTTPSHandler) Start(handlerFunc boshhandler.Func) error {
	h.dispatcher.AddRoute("/agent", h.agentHandler(handlerFunc))
	h.dispatcher.AddRoute("/blobs/", h.blobsHandler())
	return h.dispatcher.Start()
}

func (h HTTPSHandler) Stop() {
	h.dispatcher.Stop()
}

func (h HTTPSHandler) RegisterAdditionalFunc(handlerFunc boshhandler.Func) {
	panic("HTTPSHandler does not support registering additional handler funcs")
}

func (h HTTPSHandler) Send(target boshhandler.Target, topic boshhandler.Topic, message interface{}) error {
	return nil
}

func (h HTTPSHandler) agentHandler(handlerFunc boshhandler.Func) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		rawJSONPayload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			err = bosherr.WrapError(err, "Reading http body")
			h.logger.Error("https_handler", err.Error())
			return
		}

		respBytes, _, err := boshhandler.PerformHandlerWithJSON(
			rawJSONPayload,
			handlerFunc,
			boshhandler.UnlimitedResponseLength,
			h.logger,
		)
		if err != nil {
			err = bosherr.WrapError(err, "Running handler in a nice JSON sandwhich")
			h.logger.Error("https_handler", err.Error())
			return
		}

		_, err = w.Write(respBytes)
		if err != nil {
			err = bosherr.WrapError(err, "Writing response")
			h.logger.Error("https_handler", err.Error())
		}
	}
}

func (h HTTPSHandler) blobsHandler() (blobsHandler func(http.ResponseWriter, *http.Request)) {
	blobsHandler = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			h.getBlob(w, r)
		case "PUT":
			h.putBlob(w, r)
		default:
			w.WriteHeader(404)
		}
		return
	}
	return
}

func (h HTTPSHandler) putBlob(w http.ResponseWriter, r *http.Request) {
	_, blobID := path.Split(r.URL.Path)
	blobManager := blobstore.NewBlobManager(h.fs, h.dirProvider.MicroStore())

	err := blobManager.Write(blobID, r.Body)
	if err != nil {
		w.WriteHeader(500)
		if _, wErr := w.Write([]byte(err.Error())); wErr != nil {
			h.logger.Error("https_handler", "Failed to write response body: %s", wErr.Error())
		}
		return
	}

	w.WriteHeader(201)
}

func (h HTTPSHandler) getBlob(w http.ResponseWriter, r *http.Request) {
	_, blobID := path.Split(r.URL.Path)
	blobManager := blobstore.NewBlobManager(h.fs, h.dirProvider.MicroStore())

	file, err, statusCode := blobManager.Fetch(blobID)

	if err != nil {
		h.logger.Error("https_handler", "Failed to fetch blob: %s", err.Error())

		w.WriteHeader(statusCode)

	} else {
		defer func() {
			_ = file.Close()
		}()
		reader := bufio.NewReader(file)
		if _, wErr := io.Copy(w, reader); wErr != nil {
			h.logger.Error("https_handler", "Failed to write response body: %s", wErr.Error())
		}
	}
}

// Utils:

type concreteHTTPHandler struct {
	Callback func(http.ResponseWriter, *http.Request)
}

func (e concreteHTTPHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) { e.Callback(rw, r) }
