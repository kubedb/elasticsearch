package docker

const (
	ImageOperator         = "kubedb/operator"
	ImagePostgresOperator = "kubedb/pg-operator"
	ImagePostgres         = "kubedb/postgres"
	ImageMySQLOperator    = "kubedb/mysql-operator"
	ImageMySQL            = "library/mysql"
	ImageElasticOperator  = "kubedb/es-operator"
	ImageElasticsearch    = "aerokite/elasticsearch"
	ImageElasticdump      = "aerokite/elasticdump"
)

const (
	OperatorName       = "kubedb-operator"
	OperatorContainer  = "operator"
	OperatorPortName   = "web"
	OperatorPortNumber = 8080
)
