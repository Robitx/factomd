package state

import (
	"github.com/FactomProject/factomd/common/constants"
	"github.com/FactomProject/factomd/common/directoryBlock"
	"github.com/FactomProject/factomd/common/factoid/block"
	"github.com/FactomProject/factomd/common/interfaces"
	"github.com/FactomProject/factomd/common/primitives"
	"github.com/FactomProject/factomd/database/hybridDB"
	"github.com/FactomProject/factomd/log"
	"github.com/FactomProject/factomd/util"
	"os"
	"sync"
)

type State struct {
	once sync.Once
	Cfg  interfaces.IFactomConfig

	networkInMsgQueue      chan interfaces.IMsg
	networkOutMsgQueue     chan interfaces.IMsg
	networkInvalidMsgQueue chan interfaces.IMsg
	inMsgQueue             chan interfaces.IMsg
	leaderInMsgQueue       chan interfaces.IMsg
	followerInMsgQueue     chan interfaces.IMsg

	//Network MAIN = 0, TEST = 1, LOCAL = 2, CUSTOM = 3
	NetworkNumber int // Encoded into Directory Blocks

	// Number of Servers acknowledged by Factom
	TotalServers int
	ServerState  int                // (0 if client, 1 if server, 2 if audit server
	Matryoshka   []interfaces.IHash // Reverse Hash

	// Database
	DB interfaces.IDatabase

	// Directory Block State
	CurrentDirectoryBlock interfaces.IDirectoryBlock
	DBHeight              uint32

	// Web Services
	Port int

	// Message State
	LastAck interfaces.IMsg // Return the last Acknowledgement set by this server

	FactoidState      interfaces.IFactoidState
	CurrentAdminBlock interfaces.IAdminBlock
}

var _ interfaces.IState = (*State)(nil)

func (s *State) GetCurrentAdminBlock() interfaces.IAdminBlock {
	return s.CurrentAdminBlock
}

func (s *State) SetCurrentAdminBlock(adblock interfaces.IAdminBlock) {
	s.CurrentAdminBlock = adblock
}

func (s *State) GetFactoidState() interfaces.IFactoidState {
	return s.FactoidState
}

// Allow us the ability to update the port number at run time....
func (s *State) SetPort(port int) {
	// Get our factomd configuration information.
	cfg := s.GetCfg().(*util.FactomdConfig)
	cfg.Wsapi.PortNumber = port
}

func (s *State) GetPort() int {
	cfg := s.GetCfg().(*util.FactomdConfig)
	return cfg.Wsapi.PortNumber
}

// Tests the given hash, and returns true if this server is the leader for this key.
// For example, keys we test include:
//
// The Hash of the Factoid Hash
// Entry Credit Addresses
// ChainIDs
// ...
func (s *State) LeaderFor([]byte) bool {
	if s.TotalServers == 1 && s.ServerState == 1 && s.NetworkNumber == 2 {
		return true
	}
	return false
}

func (s *State) NetworkInMsgQueue() chan interfaces.IMsg {
	return s.networkInMsgQueue
}

func (s *State) NetworkInvalidMsgQueue() chan interfaces.IMsg {
	return s.networkInvalidMsgQueue
}

func (s *State) NetworkOutMsgQueue() chan interfaces.IMsg {
	return s.networkOutMsgQueue
}

func (s *State) InMsgQueue() chan interfaces.IMsg {
	return s.inMsgQueue
}

func (s *State) LeaderInMsgQueue() chan interfaces.IMsg {
	return s.leaderInMsgQueue
}

func (s *State) FollowerInMsgQueue() chan interfaces.IMsg {
	return s.followerInMsgQueue
}

//var _ IState = (*State)(nil)

// Getting the cfg state for Factom doesn't force a read of the config file unless
// it hasn;t been read yet.
func (s *State) GetCfg() interfaces.IFactomConfig {
	s.once.Do(func() {
		log.Printfln("read factom config file: %v", util.ConfigFilename())
		s.Cfg = util.ReadConfig()
	})
	return s.Cfg
}

// ReadCfg forces a read of the factom config file.  However, it does not change the
// state of any cfg object held by other processes... Only what will be returned by
// future calls to Cfg().
func (s *State) ReadCfg() interfaces.IFactomConfig {
	s.Cfg = util.ReadConfig()
	return s.Cfg
}

func (s *State) GetTotalServers() int {
	return s.TotalServers
}

func (s *State) GetServerState() int {
	return s.ServerState
}

func (s *State) GetNetworkNumber() int {
	return s.NetworkNumber
}

func (s *State) GetMatryoshka() []interfaces.IHash {
	return s.Matryoshka
}

func (s *State) GetLastAck() interfaces.IMsg {
	return s.LastAck
}

func (s *State) Init() {

	// Get our factomd configuration information.
	cfg := s.GetCfg().(*util.FactomdConfig)

	log.SetLevel(cfg.Log.ConsoleLogLevel)

	s.networkInMsgQueue = make(chan interfaces.IMsg, 10000)      //incoming message queue from the network messages
	s.networkInvalidMsgQueue = make(chan interfaces.IMsg, 10000) //incoming message queue from the network messages
	s.networkOutMsgQueue = make(chan interfaces.IMsg, 10000)     //Messages to be broadcast to the network
	s.inMsgQueue = make(chan interfaces.IMsg, 10000)             //incoming message queue for factom application messages
	s.leaderInMsgQueue = make(chan interfaces.IMsg, 10000)       //Leader Messages
	s.followerInMsgQueue = make(chan interfaces.IMsg, 10000)     //Follower Messages

	s.TotalServers = 1
	s.ServerState = 1

	//Database
	if err := s.InitBoltDB(); err != nil {
		log.Printfln("Error initializing the database: %v", err)
	}

	//Network
	switch cfg.App.Network {
	case "MAIN":
		s.NetworkNumber = constants.NETWORK_MAIN
	case "TEST":
		s.NetworkNumber = constants.NETWORK_TEST
	case "LOCAL":
		s.NetworkNumber = constants.NETWORK_LOCAL
	case "CUSTOM":
		s.NetworkNumber = constants.NETWORK_CUSTOM
	default:
		panic("Bad value for Network in factomd.conf")
	}

	s.loadDatabase()

}

func (s *State) loadDatabase() {

	dblk := new(directoryblock.DirectoryBlock)
	_, err := s.DB.Get([]byte(constants.DB_DIRECTORY_BLOCKS), constants.D_CHAINID, dblk)
	if err != nil {
		panic(err.Error())
	}

	if dblk == nil && s.NetworkNumber == constants.NETWORK_LOCAL {
		dblk, err := directoryblock.CreateDBlock(0, nil, 4)
		if err != nil {
			panic("Failed to initialize Factoids: " + err.Error())
		}

		//TODO Also need to set Admin block and EC Credit block

		fblk := block.GetGenesisFBlock()
		s.GetDB().Put([]byte(constants.DB_FACTOID_BLOCKS), primitives.Sha(constants.FACTOID_CHAINID).Bytes(), fblk)

		dblk.GetDBEntries()[2].SetKeyMR(fblk.GetKeyMR())

		s.SetCurrentDirectoryBlock(dblk)
		s.SetDBHeight(dblk.GetHeader().GetDBHeight() + 1)

		s.FactoidState = new(FactoidState)
		if err := s.FactoidState.AddTransactionBlock(fblk); err != nil {
			panic("Failed to initialize Factoids: " + err.Error())
		}
		dblk, err = directoryblock.CreateDBlock(1, dblk, 4)
	}
	s.SetDBHeight(dblk.GetHeader().GetDBHeight())
	s.SetCurrentDirectoryBlock(dblk)

}

func (s *State) InitLevelDB() error {
	cfg := s.Cfg.(*util.FactomdConfig)
	path := cfg.App.LdbPath + "/" + cfg.App.Network + "/" + "factoid_level.db"

	log.Printfln("Creating Database at %v", path)

	dbase, err := hybridDB.NewLevelMapHybridDB(path, false)

	if err != nil {
		return err
	}

	if dbase == nil {
		dbase, err = hybridDB.NewLevelMapHybridDB(path, true)
		if err != nil {
			return err
		}
	}

	//s.db = databaseOverlay.NewOverlay(dbase)
	return nil
}

func (s *State) InitBoltDB() error {
	cfg := s.Cfg.(*util.FactomdConfig)
	path := cfg.App.BoltDBPath + "/" + cfg.App.Network + "/"
	os.MkdirAll(path, 0777)
	dbase := hybridDB.NewBoltMapHybridDB(nil, path+"FactomBolt.db")
	s.DB = dbase
	return nil
}

func (s *State) String() string {
	return (s.Cfg.(*util.FactomdConfig)).String()
}

func (s *State) GetNetworkName() string {
	return (s.Cfg.(util.FactomdConfig)).App.Network

}

func (s *State) GetCurrentDirectoryBlock() interfaces.IDirectoryBlock {
	return s.CurrentDirectoryBlock
}

func (s *State) SetCurrentDirectoryBlock(dirblk interfaces.IDirectoryBlock) {
	s.CurrentDirectoryBlock = dirblk
}

func (s *State) GetDB() interfaces.IDatabase {
	return s.DB
}

func (s *State) SetDB(db interfaces.IDatabase) {
	s.DB = db
}

func (s *State) GetDBHeight() uint32 {
	return s.DBHeight
}

func (s *State) SetDBHeight(dbheight uint32) {
	s.DBHeight = dbheight
}

func (s *State) GetNewHash() interfaces.IHash {
	return new(primitives.Hash)
}
