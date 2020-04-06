package exampleproto

import (
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/jsonsrv"
)

func Negotiator(min, max int) sockerball.Negotiator {
	if min <= 0 {
		min = 1
	}
	if max <= 0 {
		max = 3
	}
	var versions = []sockerball.VersionedProtocol{}
	if min <= 1 && max >= 1 {
		versions = append(versions,
			Proto{version: 1, messageLimit: 1000000, mapper: Mapper{}, codec: SimpleCodec{}})
	}
	if min <= 2 && max >= 2 {
		versions = append(versions,
			Proto{version: 2, messageLimit: 1000000, mapper: Mapper{}, codec: jsonsrv.Codec})
	}
	if min <= 3 && max >= 3 {
		versions = append(versions,
			Proto{version: 3, messageLimit: 1000000, mapper: Mapper{}, codec: CompressingCodec{}})
	}
	return sockerball.NewVersionNegotiator(versions...)
}
