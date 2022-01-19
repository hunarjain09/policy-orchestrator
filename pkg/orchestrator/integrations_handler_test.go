package orchestrator_test

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hexa-org/policy-orchestrator/pkg/databasesupport"
	"github.com/hexa-org/policy-orchestrator/pkg/hawksupport"
	"github.com/hexa-org/policy-orchestrator/pkg/orchestrator"
	"github.com/hexa-org/policy-orchestrator/pkg/orchestrator/provider"
	"github.com/hexa-org/policy-orchestrator/pkg/websupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net"
	"net/http"
	"testing"
)

type HandlerSuite struct {
	suite.Suite
	db      *sql.DB
	server  *http.Server
	key     string
	gateway orchestrator.IntegrationsDataGateway
}

func TestIntegrationsHandler(t *testing.T) {
	suite.Run(t, &HandlerSuite{})
}

func (s *HandlerSuite) SetupTest() {
	s.db, _ = databasesupport.Open("postgres://orchestrator:orchestrator@localhost:5432/orchestrator_test?sslmode=disable")
	s.gateway = orchestrator.IntegrationsDataGateway{DB: s.db}

	// todo - move below to scenario style
	_, _ = s.db.Exec("delete from integrations;")

	listener, _ := net.Listen("tcp", "localhost:0")
	addr := listener.Addr().String()

	hash := sha256.Sum256([]byte("aKey"))
	s.key = hex.EncodeToString(hash[:])

	handlers, _ := orchestrator.LoadHandlers(s.db, hawksupport.NewCredentialStore(s.key), addr, map[string]provider.Provider{})
	s.server = websupport.Create(addr, handlers, websupport.Options{})

	go websupport.Start(s.server, listener)
	websupport.WaitForHealthy(s.server)
}

func (s *HandlerSuite) TearDownTest() {
	_ = s.db.Close()
	websupport.Stop(s.server)
}

func (s *HandlerSuite) TestList() {
	_, _ = s.gateway.Create("aName", "google cloud", []byte("aKey"))

	resp, _ := hawksupport.HawkGet(&http.Client{}, "anId", s.key, fmt.Sprintf("http://%s/integrations", s.server.Addr))
	var jsonResponse orchestrator.Integrations
	_ = json.NewDecoder(resp.Body).Decode(&jsonResponse)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), "aName", jsonResponse.Integrations[0].Name)
	assert.Equal(s.T(), "google cloud", jsonResponse.Integrations[0].Provider)
	assert.Equal(s.T(), []byte("aKey"), jsonResponse.Integrations[0].Key)
}

func (s *HandlerSuite) TestCreate() {
	integration := orchestrator.Integration{Name: "aName", Provider: "google cloud", Key: []byte("aKey")}
	marshal, _ := json.Marshal(integration)
	_, _ = hawksupport.HawkPost(&http.Client{}, "anId", s.key, fmt.Sprintf("http://%s/integrations", s.server.Addr), bytes.NewReader(marshal))

	all, _ := s.gateway.Find()
	assert.Equal(s.T(), 1, len(all))
	assert.Equal(s.T(), "aName", all[0].Name)
	assert.Equal(s.T(), "google cloud", all[0].Provider)
	assert.Equal(s.T(), []byte("aKey"), all[0].Key)
}

func (s *HandlerSuite) TestDelete() {
	id, _ := s.gateway.Create("aName", "google cloud", []byte("aKey"))

	resp, _ := hawksupport.HawkGet(&http.Client{}, "anId", s.key, fmt.Sprintf("http://%s/integrations/%s", s.server.Addr, id))
	assert.Equal(s.T(), resp.StatusCode, http.StatusOK)

	all, _ := s.gateway.Find()
	assert.Equal(s.T(), 0, len(all))
}

func (s *HandlerSuite) TestDelete_withUnknownID() {
	_, _ = s.gateway.Create("aName", "google cloud", []byte("aKey"))

	resp, _ := hawksupport.HawkGet(&http.Client{}, "anId", s.key, fmt.Sprintf("http://%s/integrations/%s", s.server.Addr, "0000"))
	assert.Equal(s.T(), resp.StatusCode, http.StatusInternalServerError)
}
