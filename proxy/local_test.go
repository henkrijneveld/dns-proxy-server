package proxy

import (
	"testing"
	"github.com/mageddo/dns-proxy-server/events/local"
	"github.com/mageddo/go-logging"
	"github.com/stretchr/testify/assert"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/mock"
	"github.com/mageddo/dns-proxy-server/cache/lru"
	hashlru "github.com/hashicorp/golang-lru"
	"fmt"
	"github.com/mageddo/dns-proxy-server/cache/store"
	"time"
)

func TestLocalDnsSolver_Solve(t *testing.T) {

	defer local.ResetConf()

	ctx := logging.NewContext()
	conf, err := local.LoadConfiguration(ctx)
	assert.Nil(t, err, "failed to load configuration")

	expectedHostname := "github.com"
	host := local.HostnameVo{Hostname: expectedHostname, Env:"", Ttl:50, Ip:[4]byte{192,168,0,1}}
	conf.AddHostname(ctx, "", host)

	question := new(dns.Question)
	question.Name = expectedHostname + "."
	solver := NewLocalDNSSolver(store.GetInstance())

	// act
	res, err := solver.Solve(ctx, *question)
	assert.Nil(t, err, "Fail to solve")

	// assert
	assert.Equal(t, 1, len(res.Answer))
	assert.Equal(t, "github.com.	0	IN	A	192.168.0.1", res.Answer[0].String())

}

func TestLocalDnsSolver_SolveNotFoundHost(t *testing.T) {

	defer local.ResetConf()

	expectedHostname := "github.com"
	ctx := logging.NewContext()

	question := new(dns.Question)
	question.Name = expectedHostname + "."
	solver := NewLocalDNSSolver(store.GetInstance())

	// act
	_, err := solver.Solve(ctx, *question)
	assert.NotNil(t, err, "Fail to solve")

}


type MockCache struct {
	mock.Mock
	lru.LRUCache
}

//
// spy put method
//
func (m *MockCache) PutIfAbsent(key, value interface{}) interface{} {
	fmt.Println("mocked!!!!!")
	m.Called(key, value)
	return m.LRUCache.PutIfAbsent(key, value)
}

//
// Testing if cache is working
// In first time must load hostname from file
// In second must load from cache
//
func TestLocalDnsSolver_SolveValidatingCache(t *testing.T) {

	defer local.ResetConf()

	ctx := logging.NewContext()
	conf, err := local.LoadConfiguration(ctx)
	assert.Nil(t, err, "failed to load configuration")

	// configuring a new host at local configuration
	expectedHostname := "github.com"
	host := local.HostnameVo{Hostname: expectedHostname, Env:"", Ttl:50, Ip:[4]byte{192,168,0,1}}
	conf.AddHostname(ctx, "", host)

	// creating a request for the created host
	question := new(dns.Question)
	question.Name = expectedHostname + "."

	// stubbing cache to verify the calls
	mockCache := &MockCache{}
	mockCache.Cache, err = hashlru.New(1)
	assert.Nil(t, err, "Failed to create cache")
	mockCache.On("PutIfAbsent", expectedHostname, mock.Anything).Once()

	solver := NewLocalDNSSolver(mockCache)

	// we ask for the same host 5 times but it must load from file just once
	for i:=5; i > 0; i-- {

		// act
		res, err := solver.Solve(ctx, *question)
		assert.Nil(t, err, "Fail to solve")

		// assert
		assert.Equal(t, 1, len(res.Answer))
		assert.Equal(t, "github.com.	0	IN	A	192.168.0.1", res.Answer[0].String())

	}

	mockCache.AssertExpectations(t)

}


func TestLocalDnsSolver_Solve_CacheExpiration(t *testing.T) {

	defer local.ResetConf()

	ctx := logging.NewContext()
	conf, err := local.LoadConfiguration(ctx)
	assert.Nil(t, err, "failed to load configuration")

	// configuring a new host at local configuration
	expectedHostname := "github.com"
	host := local.HostnameVo{Hostname: expectedHostname, Env:"", Ttl:2, Ip:[4]byte{192,168,0,1}}
	conf.AddHostname(ctx, "", host)

	// creating a request for the created host
	question := new(dns.Question)
	question.Name = expectedHostname + "."

	// stubbing cache to verify the calls
	mockCache := &MockCache{}
	mockCache.Cache, err = hashlru.New(1)
	assert.Nil(t, err, "Failed to create cache")
	mockCache.On("PutIfAbsent", expectedHostname, mock.Anything).Twice()

	solver := NewLocalDNSSolver(mockCache)

	// we ask for the same host 5 times but it must load from file just once
	for i:=3; i > 0; i-- {

		time.Sleep(time.Duration(int64(1100)) * time.Millisecond)

		// act
		res, err := solver.Solve(ctx, *question)
		assert.Nil(t, err, "Fail to solve")

		// assert
		assert.Equal(t, 1, len(res.Answer))
		assert.Equal(t, "github.com.	0	IN	A	192.168.0.1", res.Answer[0].String())

	}

	mockCache.AssertExpectations(t)

}
