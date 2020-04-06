package exampleproto

import (
	"github.com/shabbyrobe/sockerball"
)

type Proto struct {
	version      int
	name         string
	mapper       sockerball.Mapper
	codec        sockerball.Codec
	messageLimit int
}

func (p Proto) ProtocolName() string      { return p.name }
func (p Proto) Mapper() sockerball.Mapper { return p.mapper }
func (p Proto) Codec() sockerball.Codec   { return p.codec }
func (p Proto) Version() int              { return p.version }
func (p Proto) MessageLimit() int         { return p.messageLimit }
