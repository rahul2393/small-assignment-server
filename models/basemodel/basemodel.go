package basemodel

import (
	"gopkg.in/gorp.v1"
)

type BaseModel struct {
	// ID is the auto incremented primary key.
	ID int64 `json:"id"`
	// Created is the server creation timestamp in milliseconds.
	Created int64 `json:"created"`
	// Updated is the server updated timestamp in milliseconds.
	Updated int64 `json:"updated"`
	// Deleted is a bool flag to indicate deletion.
	Deleted bool `json:"deleted"`
	// TableName is the name of the model.
	ModelName string `db:"-" json:"modelName"`
}

type TableNamer interface {
	TableName() string
}

type Model interface {
	TableNamer
	Deleter
}

type Expander interface {
	Expand(s gorp.SqlExecutor, exclude string) error
}

type Deleter interface {
	Delete(s gorp.SqlExecutor) error
}
