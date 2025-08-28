package mock

import (
	"reflect"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/translator"
)

// Implementation for the mock model.
//
// Its generation behavior can be configured using the [AddDropIn] and [SetConfig] methods.
type Implementation struct {
	dropIns translator.DropIns
	conf    config.Config
}

// GetDropIns returns the drop ins passed to the [Implementation] struct.
func (m Implementation) GetDropIns() translator.DropIns {
	return m.dropIns
}

// GetOpenCypherConfig returns the config passed to the [Implementation] struct.
func (m Implementation) GetOpenCypherConfig() config.Config {
	return m.conf
}

// AddDropIn registers a drop in for the implementation, which is returned when
// its [Implementation.GetDropIns] method gets called.
func (m *Implementation) AddDropIn(c translator.Clause, d translator.DropIn) {
	if m.dropIns == nil {
		m.dropIns = make(translator.DropIns)
	}
	m.dropIns[reflect.TypeOf(c)] = d
}

// SetConfig sets the implementation's config, which is returned when its
// [Implementation.GetOpenCypherConfig] method gets called.
func (m *Implementation) SetConfig(c config.Config) {
	m.conf = c
}
