package httpserver

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDomainInGrants(t *testing.T) {
	id1 := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	assert.True(t, domainInGrants([]uuid.UUID{id1, id2}, id1))
	assert.False(t, domainInGrants([]uuid.UUID{id1}, id2))
	assert.False(t, domainInGrants(nil, id1))
}
