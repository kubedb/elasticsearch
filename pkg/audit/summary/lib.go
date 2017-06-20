package summary

import (
	"fmt"
	"log"

	elastic "gopkg.in/olivere/elastic.v3"
)

func newClient(host, port string) (*elastic.Client, error) {
	return elastic.NewClient(
		elastic.SetURL(fmt.Sprintf("http://%v:%v", host, port)),
		elastic.SetMaxRetries(10),
		elastic.SetSniff(false),
	), nil
}

func getAllIndices(client *elastic.Client) ([]string, error) {
	return client.IndexNames()
}

func getDataFromIndex(client *elastic.Client, indexName string) *ElasticData {
	// Get analyzer
	analyzerData, err := client.IndexGetSettings(indexName).Do()
	if err != nil {
		log.Fatal(err)
	}

	dataByte, err := json.Marshal(analyzerData[indexName].Settings["index"])
	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(dataByte, &elasticData.Setting); err != nil {
		log.Fatal(err)
	}

	// get mappings
	mappingData, err := client.GetMapping().Index(indexName).Do()
	if err != nil {
		log.Fatal(err)
	}
	elasticData.Mapping = mappingData

	// Count Ids
	mappingDataBype, err := json.Marshal(mappingData[indexName])
	if err != nil {
		log.Fatal(err)
	}
	type esTypes struct {
		Mappings map[string]interface{} `json:"mappings"`
	}
	var esType esTypes
	if err := json.Unmarshal(mappingDataBype, &esType); err != nil {
		log.Fatal(err)
	}
	for key := range esType.Mappings {
		counts, err := client.Count(indexName).Type(key).Do()
		if err != nil {
			log.Fatal(err)
		}
		elasticData.IdCount[key] = counts
	}
	return elasticData
}
