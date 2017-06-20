package summary

import (
	"fmt"
	"net/http"

	tcs "github.com/k8sdb/apimachinery/client/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	clientset "k8s.io/client-go/kubernetes"
	"github.com/k8sdb/elasticsearch/pkg/audit/type"
)

func GetSummaryReport(
	kubeClient clientset.Interface,
	dbClient tcs.ExtensionInterface,
	namespace string,
	kubedbName string,
	index string,
	w http.ResponseWriter,
) {

	if _, err := dbClient.Elastics(namespace).Get(kubedbName); err != nil {
		if kerr.IsNotFound(err) {
			http.Error(w, fmt.Sprintf(`Elastic "%v" not found`, kubedbName), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	host := fmt.Sprintf("%v.%v", kubedbName, namespace)
	port := "9200"

	client, err := newClient(host, port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	indices := make([]string, 0)
	if index == "" {
		indices, err = GetAllIndices(client)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		indices = append(indices, index)
	}

	infos := make(map[string]*types.IndexInfo)
	for _, index := range indices {
		lib.DumpDBInfo(client, req.Dir, index)
	}

	if data != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, string(data))
	} else {
		http.Error(w, "audit data not found", http.StatusNotFound)
	}
}
