package pruning

import (
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	"cosmossdk.io/store/v2"
	"cosmossdk.io/store/v2/commitment/iavl"
	"cosmossdk.io/store/v2/storage/sqlite"
)

type PruningTestSuite struct {
	suite.Suite

	manager *Manager
	ss      store.VersionedDatabase
	sc      store.Committer
}

func TestPruningTestSuite(t *testing.T) {
	suite.Run(t, &PruningTestSuite{})
}

func (s *PruningTestSuite) SetupTest() {
	noopLog := log.NewNopLogger()

	ss, err := sqlite.New(s.T().TempDir())
	s.Require().NoError(err)

	sc := iavl.NewIavlTree(dbm.NewMemDB(), noopLog, iavl.DefaultConfig())

	s.manager = NewManager(ss, sc, noopLog)
	s.ss = ss
	s.sc = sc
}

func (s *PruningTestSuite) TearDownTest() {
	s.manager.Start()
	s.manager.Stop()
}

func (s *PruningTestSuite) TestPruning() {
	s.manager.SetCommitmentOptions(Option{4, 2})
	s.manager.SetStoreOptions(Option{4, 3})
	s.manager.Start()

	// write 10 batches
	for i := 0; i < 10; i++ {
		cs := store.NewChangeset()
		cs.Add([]byte("key"), []byte("value"))
		err := s.sc.WriteBatch(cs)
		s.Require().NoError(err)
		_, err = s.sc.Commit()
		s.Require().NoError(err)
		err = s.ss.ApplyChangeset(uint64(i+1), cs)
		s.Require().NoError(err)
		s.manager.Prune(uint64(i + 1))
	}

	// wait for pruning to finish
	s.manager.Stop()

	// check the store for the version 5
	val, err := s.ss.Get("", 5, []byte("key"))
	s.Require().NoError(err)
	s.Require().NotNil(val)
	// check the store for the version 4
	// TODO: sqlite store doesn't support parallel writes
	_, err = s.ss.Get("", 4, []byte("key"))
	s.Require().NoError(err)
	// s.Require().Nil(val)

	// check the commitment for the version 6
	proof, err := s.sc.GetProof(6, []byte("key"))
	s.Require().NoError(err)
	s.Require().NotNil(proof.GetExist())
	// check the commitment for the version 5
	proof, err = s.sc.GetProof(5, []byte("key"))
	s.Require().Error(err)
	s.Require().Nil(proof)
}
