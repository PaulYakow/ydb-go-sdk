package cluster

import (
	"container/list"
	"context"
	"fmt"
	cluster2 "github.com/ydb-platform/ydb-go-sdk/v3/cluster"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/driver/cluster/balancer"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/driver/cluster/balancer/conn"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/driver/cluster/balancer/conn/entry"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/driver/cluster/balancer/conn/info"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/driver/cluster/balancer/conn/runtime/stats"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/driver/cluster/balancer/conn/runtime/stats/state"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/ydb-platform/ydb-go-sdk/v3/internal/timeutil"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/timeutil/timetest"
)

func TestClusterFastRedial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := newStubListener()
	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(ln)
	}()

	cs, balancer := simpleBalancer()
	c := &cluster{
		dial: func(ctx context.Context, s string, p int) (conn.Conn, error) {
			cc, err := ln.Dial(ctx)
			return &conn.conn{
				addr: cluster2.Addr{s, p},
				raw:  cc,
			}, err
		},
		balancer: balancer,
	}

	pingConnects := func(size int) chan struct{} {
		done := make(chan struct{})
		go func() {
			for i := 0; i < size*10; i++ {
				con, err := c.Get(context.Background())
				// enforce close bad connects to track them
				if err == nil && con != nil && con.addr.addr == "bad" {
					_ = con.raw.Close()
				}
			}
			close(done)
		}()
		return done
	}

	ne := []cluster2.Endpoint{
		{Host: "foo"},
		{Host: "bad"},
	}
	mergeEndpointIntoCluster(ctx, c, []cluster2.Endpoint{}, ne)
	select {
	case <-pingConnects(len(ne)):

	case <-time.After(time.Second * 10):
		t.Fatalf("Time limit exceeded while %d endpoints in balance. Wait channel used", len(*cs))
	}
}

func withDisabledTrackerQueue(c *cluster) *cluster {
	c.index = make(map[cluster2.Addr]entry.Entry)
	c.trackerQueue = list.New()
	c.once.Do(func() {})
	return c
}

func TestClusterMergeEndpoints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := newStubListener()
	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(ln)
	}()

	cs, balancer := simpleBalancer()
	c := withDisabledTrackerQueue(&cluster{
		dial: func(ctx context.Context, s string, p int) (conn.Conn, error) {
			cc, err := ln.Dial(ctx)
			return &conn.conn{
				addr: cluster2.Addr{s, p},
				raw:  cc,
			}, err
		},
		balancer: balancer,
	})

	pingConnects := func(size int) {
		for i := 0; i < size*10; i++ {
			sub, cancel := context.WithTimeout(ctx, time.Millisecond)
			defer cancel()
			con, err := c.Get(sub)
			// enforce close bad connects to track them
			if err == nil && con != nil && con.addr.addr == "bad" {
				_ = con.raw.Close()
			}
		}
	}
	assert := func(t *testing.T, total, inBalance, onTracking int) {
		if len(c.index) != total {
			t.Fatalf("total expected number of endpoints %d got %d", total, len(c.index))
		}
		if len(*cs) != inBalance {
			t.Fatalf("inBalance expected number of endpoints %d got %d", inBalance, len(*cs))
		}
		if c.trackerQueue.Len() != onTracking {
			t.Fatalf("onTracking expected number of endpoints %d got %d", onTracking, c.trackerQueue.Len())
		}
	}

	endpoints := []cluster2.Endpoint{
		{Host: "foo"},
		{Host: "foo", Port: 123},
	}
	badEndpoints := []cluster2.Endpoint{
		{Host: "bad"},
		{Host: "bad", Port: 123},
	}
	nextEndpoints := []cluster2.Endpoint{
		{Host: "foo"},
		{Host: "bar"},
		{Host: "bar", Port: 123},
	}
	nextBadEndpoints := []cluster2.Endpoint{
		{Host: "bad", Port: 23},
	}
	t.Run("initial fill", func(t *testing.T) {
		ne := append(endpoints, badEndpoints...)
		// merge new endpoints into balancer
		mergeEndpointIntoCluster(ctx, c, []cluster2.Endpoint{}, ne)
		// try endpoints, filter out bad ones to tracking
		pingConnects(len(ne))
		assert(t, len(ne), len(endpoints), len(badEndpoints))
	})
	t.Run("update with another endpoints", func(t *testing.T) {
		ne := append(nextEndpoints, nextBadEndpoints...)
		// merge new endpoints into balancer
		mergeEndpointIntoCluster(ctx, c, append(endpoints, badEndpoints...), ne)
		// try endpoints, filter out bad ones to tracking
		pingConnects(len(ne))
		assert(t, len(ne), len(nextEndpoints), len(nextBadEndpoints))
	})
	t.Run("left only bad", func(t *testing.T) {
		ne := nextBadEndpoints
		// merge new endpoints into balancer
		mergeEndpointIntoCluster(ctx, c, append(nextEndpoints, nextBadEndpoints...), ne)
		// try endpoints, filter out bad ones to tracking
		pingConnects(len(ne))
		assert(t, len(ne), 0, len(nextBadEndpoints))
	})
	t.Run("left only good", func(t *testing.T) {
		ne := nextEndpoints
		// merge new endpoints into balancer
		mergeEndpointIntoCluster(ctx, c, nextBadEndpoints, ne)
		// try endpoints, filter out bad ones to tracking
		pingConnects(len(ne))
		assert(t, len(ne), len(nextEndpoints), 0)
	})
}

func TestClusterRemoveTracking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := newStubListener()
	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(ln)
	}()

	_, balancer := simpleBalancer()

	// Prevent tracker timer from firing.
	timer := timetest.StubSingleTimer(t)
	defer timer.Cleanup()

	tracking := make(chan int)
	defer close(tracking)
	assertTracking := func(exp int) {
		// Force tracker to collect the connections to track.
		timer.C <- timeutil.Now()
		if act := <-tracking; act != exp {
			t.Fatalf(
				"unexpected number of conns to track: %d; want %d",
				act, exp,
			)
		}
	}

	c := &cluster{
		dial: func(ctx context.Context, s string, p int) (conn.Conn, error) {
			cc, err := ln.Dial(ctx)
			return &conn.conn{
				addr: cluster2.Addr{s, p},
				raw:  cc,
			}, err
		},
		balancer: balancer,
		testHookTrackerQueue: func(q []*list.Element) {
			tracking <- len(q)
		},
	}

	endpoint := cluster2.Endpoint{Host: "foo"}
	c.Insert(ctx, endpoint)

	// Await for connection to be established.
	// Note that this is server side half.
	conn := <-ln.S

	// Do not accept new connections.
	_ = ln.Close()
	// Force cluster to reconnect.
	_ = conn.Close()
	// Await for Conn change its state inside cluster.
	{
		sub, cancel := context.WithTimeout(ctx, time.Millisecond)
		defer cancel()
		for {
			_, err := c.Get(sub)
			if err != nil {
				break
			}
		}
	}
	<-timer.Reset

	assertTracking(1)
	<-timer.Reset

	c.Remove(ctx, endpoint)

	assertTracking(0)
}

func TestClusterRemoveOffline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, balancer := simpleBalancer()

	// Prevent tracker timer from firing.
	timer := timetest.StubSingleTimer(t)
	defer timer.Cleanup()

	tracking := make(chan int)
	defer close(tracking)

	c := &cluster{
		dial: func(ctx context.Context, s string, p int) (conn.Conn, error) {
			return nil, fmt.Errorf("refused")
		},
		balancer: balancer,
		testHookTrackerQueue: func(q []*list.Element) {
			tracking <- len(q)
		},
	}

	endpoint := cluster2.Endpoint{Host: "foo"}
	c.Insert(ctx, endpoint)
	<-timer.Reset

	c.Remove(ctx, endpoint)

	timer.C <- timeutil.Now()
	if n := <-tracking; n != 0 {
		t.Fatalf("unexpected %d tracking connection(s)", n)
	}
}

func TestClusterRemoveAndInsert(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := newStubListener()
	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(ln)
	}()

	_, balancer := simpleBalancer()

	// Prevent tracker timer from firing.
	timer := timetest.StubSingleTimer(t)
	defer timer.Cleanup()

	tracking := make(chan (<-chan int))
	defer close(tracking)
	assertTracking := func(exp int) {
		// Force tracker to collect the connections to track.
		timer.C <- timeutil.Now()
		ch := <-tracking
		if act := <-ch; act != exp {
			t.Fatalf(
				"unexpected number of conns to track: %d; want %d",
				act, exp,
			)
		}
	}

	dialTicket := make(chan uint64, 1)
	c := &cluster{
		dial: func(ctx context.Context, s string, p int) (conn.Conn, error) {
			var id uint64
			select {
			case id = <-dialTicket:
			default:
				return nil, fmt.Errorf("refused")
			}
			cc, err := ln.Dial(ctx)
			ret := conn2.NewConn(cc, cluster2.Addr{s, p})
			// Used to distinguish connections.
			ret.Runtime().SetOpStarted(id)
			return ret, err
		},
		balancer: balancer,
		testHookTrackerQueue: func(q []*list.Element) {
			ch := make(chan int)
			tracking <- ch
			ch <- len(q)
		},
	}
	defer func() {
		err := c.Close()
		if err != nil {
			t.Errorf("close failed: %v", err)
		}
	}()

	t.Run("test actual block of tracker", func(t *testing.T) {
		endpoint := cluster2.Endpoint{Host: "foo"}
		c.Insert(ctx, endpoint)

		// Wait for connection become tracked.
		<-timer.Reset
		assertTracking(1)
		<-timer.Reset

		// Now force tracker to make another iteration, but not release
		// testHookTrackerQueue by reading from tracking channel.
		timer.C <- timeutil.Now()
		blocked := <-tracking

		// While our tracker is in progress (stuck on writing to the tracking
		// channel actually) remove endpoint.
		c.Remove(ctx, endpoint)

		// Now insert back the same endpoint with alive connection (and let dialer
		// to dial successfully).
		dialTicket <- 100
		c.Insert(ctx, endpoint)

		// Release the tracker iteration.
		dialTicket <- 200
		<-blocked
		<-timer.Reset
		assertTracking(0)

		var ss []stats.Stats
		c.Stats(func(_ endpoint.Endpoint, s stats.Stats) {
			ss = append(ss, s)
		})
		if len(ss) != 1 {
			t.Fatalf("unexpected number of connection stats")
		}
		if ss[0].OpStarted != 100 {
			t.Fatalf("unexpected connection used")
		}
	})
}

func TestClusterAwait(t *testing.T) {
	const timeout = 100 * time.Millisecond

	ln := newStubListener()
	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(ln)
	}()

	var connToReturn conn.Conn
	c := &cluster{
		dial: func(ctx context.Context, _ string, _ int) (_ conn.Conn, err error) {
			cc, err := ln.Dial(ctx)
			if err != nil {
				return nil, err
			}
			return &conn.conn{
				raw: cc,
			}, nil
		},
		balancer: stubBalancer{
			OnInsert: func(c conn.Conn, _ info.Info) balancer.Element {
				connToReturn = c
				return c.addr
			},
			OnNext: func() conn.Conn {
				return connToReturn
			},
		},
	}
	get := func() (<-chan error, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		got := make(chan error)
		go func() {
			_, err := c.Get(ctx)
			got <- err
		}()
		return got, cancel
	}
	{
		got, cancel := get()
		cancel()
		assertRecvError(t, timeout, got, context.Canceled)
	}
	{
		got, cancel := get()
		defer cancel()
		assertRecvError(t, timeout, got, ErrClusterEmpty)
	}
	{
		c.Insert(context.Background(), cluster2.Endpoint{})
		got, cancel := get()
		defer cancel()
		assertRecvError(t, timeout, got, nil)
	}
}

type stubBalancer struct {
	OnNext      func() conn.Conn
	OnInsert    func(conn.Conn, info.Info) balancer.Element
	OnUpdate    func(balancer.Element, info.Info)
	OnRemove    func(balancer.Element)
	OnPessimize func(balancer.Element) error
	OnContains  func(balancer.Element) bool
}

func simpleBalancer() (*list2.List, balancer.Balancer) {
	cs := new(list2.List)
	var i int
	return cs, stubBalancer{
		OnNext: func() conn.Conn {
			n := len(*cs)
			if n == 0 {
				return nil
			}
			e := (*cs)[i%n]
			i++
			return e.conn
		},
		OnInsert: func(conn conn.Conn, info info.Info) balancer.Element {
			return cs.Insert(conn, info)
		},
		OnRemove: func(x balancer.Element) {
			e := x.(*list2.Element)
			cs.Remove(e)
		},
		OnUpdate: func(x balancer.Element, info info.Info) {
			e := x.(*list2.Element)
			e.info = info
		},
		OnPessimize: func(x balancer.Element) error {
			e := x.(*list2.Element)
			e.conn.runtime.setState(state.Banned)
			return nil
		},
		OnContains: func(x balancer.Element) bool {
			e := x.(*list2.Element)
			return cs.Contains(e)
		},
	}
}

func (s stubBalancer) Next() conn.Conn {
	if f := s.OnNext; f != nil {
		return f()
	}
	return nil
}
func (s stubBalancer) Insert(c conn.Conn, i info.Info) balancer.Element {
	if f := s.OnInsert; f != nil {
		return f(c, i)
	}
	return nil
}
func (s stubBalancer) Update(el balancer.Element, i info.Info) {
	if f := s.OnUpdate; f != nil {
		f(el, i)
	}
}
func (s stubBalancer) Remove(el balancer.Element) {
	if f := s.OnRemove; f != nil {
		f(el)
	}
}
func (s stubBalancer) Pessimize(el balancer.Element) error {
	if f := s.OnPessimize; f != nil {
		return f(el)
	}
	return nil
}

func (s stubBalancer) Contains(el balancer.Element) bool {
	if f := s.OnContains; f != nil {
		return f(el)
	}
	return false
}

type stubListener struct {
	C chan net.Conn // Client half of the connection.
	S chan net.Conn // Server half of the connection.

	once sync.Once
	exit chan struct{}
}

func newStubListener() *stubListener {
	return &stubListener{
		C: make(chan net.Conn),
		S: make(chan net.Conn, 1),

		exit: make(chan struct{}),
	}
}

func (ln *stubListener) Accept() (net.Conn, error) {
	s, c := net.Pipe()
	select {
	case ln.C <- c:
	case <-ln.exit:
		return nil, fmt.Errorf("closed")
	}
	select {
	case ln.S <- s:
	default:
	}
	return s, nil
}

func (ln *stubListener) Addr() net.Addr {
	return &net.TCPAddr{}
}

func (ln *stubListener) Close() error {
	ln.once.Do(func() {
		close(ln.exit)
	})
	return nil
}

func (ln *stubListener) Dial(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			select {
			case <-ln.exit:
				return nil, fmt.Errorf("refused")
			case c := <-ln.C:
				return c, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    time.Second,
			Timeout: time.Second,
		}),
	)
}

func assertRecvError(t *testing.T, d time.Duration, e <-chan error, exp error) {
	select {
	case act := <-e:
		if act != exp {
			t.Errorf("%s: unexpected error: %v; want %v", ydb.fileLine(2), act, exp)
		}
	case <-time.After(d):
		t.Errorf("%s: nothing received after %s", ydb.fileLine(2), d)
	}
}

func mergeEndpointIntoCluster(ctx context.Context, c *cluster, curr, next []cluster2.Endpoint) {
	SortEndpoints(curr)
	SortEndpoints(next)
	DiffEndpoints(curr, next,
		func(i, j int) { c.Update(ctx, next[j]) },
		func(i, j int) { c.Insert(ctx, next[j]) },
		func(i, j int) { c.Remove(ctx, curr[i]) },
	)
}

func TestDiffEndpoint(t *testing.T) {
	// lists must be sorted
	noEndpoints := []cluster2.Endpoint{}
	someEndpoints := []cluster2.Endpoint{
		{
			Host: "0",
			Port: 0,
		},
		{
			Host: "1",
			Port: 1,
		},
	}
	sameSomeEndpoints := []cluster2.Endpoint{
		{
			Host:       "0",
			Port:       0,
			LoadFactor: 1,
			Local:      true,
		},
		{
			Host:       "1",
			Port:       1,
			LoadFactor: 2,
			Local:      true,
		},
	}
	anotherEndpoints := []cluster2.Endpoint{
		{
			Host: "2",
			Port: 0,
		},
		{
			Host: "3",
			Port: 1,
		},
	}
	moreEndpointsOverlap := []cluster2.Endpoint{
		{
			Host:       "0",
			Port:       0,
			LoadFactor: 1,
			Local:      true,
		},
		{
			Host: "1",
			Port: 1,
		},
		{
			Host: "1",
			Port: 2,
		},
	}

	type TC struct {
		name         string
		curr, next   []cluster2.Endpoint
		eq, add, del int
	}

	tests := []TC{
		{
			name: "none",
			curr: noEndpoints,
			next: noEndpoints,
			eq:   0,
			add:  0,
			del:  0,
		},
		{
			name: "equals",
			curr: someEndpoints,
			next: sameSomeEndpoints,
			eq:   2,
			add:  0,
			del:  0,
		},
		{
			name: "noneToSome",
			curr: noEndpoints,
			next: someEndpoints,
			eq:   0,
			add:  2,
			del:  0,
		},
		{
			name: "SomeToNone",
			curr: someEndpoints,
			next: noEndpoints,
			eq:   0,
			add:  0,
			del:  2,
		},
		{
			name: "SomeToMore",
			curr: someEndpoints,
			next: moreEndpointsOverlap,
			eq:   2,
			add:  1,
			del:  0,
		},
		{
			name: "MoreToSome",
			curr: moreEndpointsOverlap,
			next: someEndpoints,
			eq:   2,
			add:  0,
			del:  1,
		},
		{
			name: "SomeToAnother",
			curr: someEndpoints,
			next: anotherEndpoints,
			eq:   0,
			add:  2,
			del:  2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eq, add, del := 0, 0, 0
			DiffEndpoints(tc.curr, tc.next,
				func(i, j int) { eq++ },
				func(i, j int) { add++ },
				func(i, j int) { del++ },
			)
			if eq != tc.eq || add != tc.add || del != tc.del {
				t.Errorf("Got %d, %d, %d expected: %d, %d, %d", eq, add, del, tc.eq, tc.add, tc.del)
			}
		})
	}
}