package mongo

type Repository struct {
	URI    string
	DBName string
}

func NewRepository(uri string, dbName string) *Repository {
	return &Repository{
		URI:    uri,
		DBName: dbName,
	}
}
