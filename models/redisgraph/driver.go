/*
Package redisgraph provides the model for RedisGraph, a Redis module.
*/
package redisgraph

import (
	"github.com/Anon10214/dinkel/models/falkordb"
)

// Driver for RedisGraph
type Driver struct{ falkordb.Driver }
