package network

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pactus-project/pactus/types/amount"
	"github.com/pagu-project/Pagu/client"
	"github.com/pagu-project/Pagu/engine/command"
	"github.com/pagu-project/Pagu/utils"
)

const (
	CommandName         = "network"
	NodeInfoCommandName = "node-info"
	StatusCommandName   = "status"
	HealthCommandName   = "health"
	HelpCommandName     = "help"
)

type Network struct {
	ctx       context.Context
	clientMgr *client.Mgr
}

func NewNetwork(ctx context.Context,
	clientMgr *client.Mgr,
) Network {
	return Network{
		ctx:       ctx,
		clientMgr: clientMgr,
	}
}

type NodeInfo struct {
	PeerID              string
	IPAddress           string
	Agent               string
	Moniker             string
	Country             string
	City                string
	RegionName          string
	TimeZone            string
	ISP                 string
	ValidatorNum        int32
	AvailabilityScore   float64
	StakeAmount         int64
	LastBondingHeight   uint32
	LastSortitionHeight uint32
}

type NetStatus struct {
	NetworkName         string
	ConnectedPeersCount uint32
	ValidatorsCount     int32
	TotalBytesSent      uint32
	TotalBytesReceived  uint32
	CurrentBlockHeight  uint32
	TotalNetworkPower   int64
	TotalCommitteePower int64
	TotalAccounts       int32
	CirculatingSupply   int64
}

func (n *Network) GetCommand() command.Command {
	subCmdNodeInfo := command.Command{
		Name: NodeInfoCommandName,
		Desc: "View the information of a node",
		Help: "Provide your validator address on the specific node to get the validator and node info",
		Args: []command.Args{
			{
				Name:     "validator_address",
				Desc:     "Your validator address",
				Optional: false,
			},
		},
		SubCommands: nil,
		AppIDs:      command.AllAppIDs(),
		Handler:     n.nodeInfoHandler,
	}

	subCmdHealth := command.Command{
		Name:        HealthCommandName,
		Desc:        "Checking network health status",
		Help:        "",
		Args:        []command.Args{},
		SubCommands: nil,
		AppIDs:      command.AllAppIDs(),
		Handler:     n.networkHealthHandler,
	}

	subCmdStatus := command.Command{
		Name:        StatusCommandName,
		Desc:        "Network statistics",
		Help:        "",
		Args:        []command.Args{},
		SubCommands: nil,
		AppIDs:      command.AllAppIDs(),
		Handler:     n.networkStatusHandler,
	}

	cmdNetwork := command.Command{
		Name:        CommandName,
		Desc:        "Network related commands",
		Help:        "",
		Args:        nil,
		AppIDs:      command.AllAppIDs(),
		SubCommands: make([]command.Command, 0),
		Handler:     nil,
	}

	cmdNetwork.AddSubCommand(subCmdHealth)
	cmdNetwork.AddSubCommand(subCmdNodeInfo)
	cmdNetwork.AddSubCommand(subCmdStatus)

	return cmdNetwork
}

func (n *Network) networkHealthHandler(cmd command.Command, _ command.AppID, _ string, _ ...string) command.CommandResult {
	lastBlockTime, lastBlockHeight := n.clientMgr.GetLastBlockTime()
	lastBlockTimeFormatted := time.Unix(int64(lastBlockTime), 0).Format("02/01/2006, 15:04:05")
	currentTime := time.Now()

	timeDiff := (currentTime.Unix() - int64(lastBlockTime))

	healthStatus := true
	if timeDiff > 15 {
		healthStatus = false
	}

	var status string
	if healthStatus {
		status = "Healthy✅"
	} else {
		status = "UnHealthy❌"
	}

	return cmd.SuccessfulResult("Network is %s\nCurrentTime: %v\nLastBlockTime: %v\nTime Diff: %v\nLast Block Height: %v",
		status, currentTime.Format("02/01/2006, 15:04:05"), lastBlockTimeFormatted, timeDiff, utils.FormatNumber(int64(lastBlockHeight)))
}

func (be *Network) networkStatusHandler(cmd command.Command, _ command.AppID, _ string, _ ...string) command.CommandResult {
	netInfo, err := be.clientMgr.GetNetworkInfo()
	if err != nil {
		return cmd.ErrorResult(err)
	}

	chainInfo, err := be.clientMgr.GetBlockchainInfo()
	if err != nil {
		return cmd.ErrorResult(err)
	}

	cs, err := be.clientMgr.GetCirculatingSupply()
	if err != nil {
		cs = 0
	}

	// Convert NanoPAC to PAC using the Amount type.
	totalNetworkPower := amount.Amount(chainInfo.TotalPower).ToPAC()
	totalCommitteePower := amount.Amount(chainInfo.CommitteePower).ToPAC()
	circulatingSupply := amount.Amount(cs).ToPAC()

	net := NetStatus{
		ValidatorsCount:     chainInfo.TotalValidators,
		CurrentBlockHeight:  chainInfo.LastBlockHeight,
		TotalNetworkPower:   int64(totalNetworkPower),
		TotalCommitteePower: int64(totalCommitteePower),
		NetworkName:         netInfo.NetworkName,
		TotalAccounts:       chainInfo.TotalAccounts,
		CirculatingSupply:   int64(circulatingSupply),
	}

	return cmd.SuccessfulResult("Network Name: %s\nConnected Peers: %v\n"+
		"Validators Count: %v\nAccounts Count: %v\nCurrent Block Height: %v\nTotal Power: %v PAC\nTotal Committee Power: %v PAC\nCirculating Supply: %v PAC\n"+
		"\n> Note📝: This info is from one random network node. Non-blockchain data may not be consistent.",
		net.NetworkName,
		utils.FormatNumber(int64(net.ConnectedPeersCount)),
		utils.FormatNumber(int64(net.ValidatorsCount)),
		utils.FormatNumber(int64(net.TotalAccounts)),
		utils.FormatNumber(int64(net.CurrentBlockHeight)),
		utils.FormatNumber(net.TotalNetworkPower),
		utils.FormatNumber(net.TotalCommitteePower),
		utils.FormatNumber(net.CirculatingSupply),
	)
}

func (n *Network) nodeInfoHandler(cmd command.Command, _ command.AppID, _ string, args ...string) command.CommandResult {
	valAddress := args[0]

	peerInfo, err := n.clientMgr.GetPeerInfo(valAddress)
	if err != nil {
		return cmd.ErrorResult(err)
	}

	peerID, err := peer.IDFromBytes(peerInfo.PeerId)
	if err != nil {
		return cmd.ErrorResult(err)
	}

	ip := utils.ExtractIPFromMultiAddr(peerInfo.Address)
	geoData := utils.GetGeoIP(ip)

	nodeInfo := &NodeInfo{
		PeerID:     peerID.String(),
		IPAddress:  peerInfo.Address,
		Agent:      peerInfo.Agent,
		Moniker:    peerInfo.Moniker,
		Country:    geoData.CountryName,
		City:       geoData.City,
		RegionName: geoData.RegionName,
		TimeZone:   geoData.TimeZone,
		ISP:        geoData.ISP,
	}

	// here we check if the node is also a validator.
	// if its a validator , then we populate the validator data.
	// if not validator then we set everything to 0/empty .
	val, err := n.clientMgr.GetValidatorInfo(valAddress)
	if err == nil && val != nil {
		nodeInfo.ValidatorNum = val.Validator.Number
		nodeInfo.AvailabilityScore = val.Validator.AvailabilityScore
		// Convert NanoPAC to PAC using the Amount type and then to int64.
		stakeAmount := amount.Amount(val.Validator.Stake).ToPAC()
		nodeInfo.StakeAmount = int64(stakeAmount) // Convert float64 to int64.
		nodeInfo.LastBondingHeight = val.Validator.LastBondingHeight
		nodeInfo.LastSortitionHeight = val.Validator.LastSortitionHeight
	} else {
		nodeInfo.ValidatorNum = 0
		nodeInfo.AvailabilityScore = 0
		nodeInfo.StakeAmount = 0
		nodeInfo.LastBondingHeight = 0
		nodeInfo.LastSortitionHeight = 0
	}

	var pip19Score string
	if nodeInfo.AvailabilityScore >= 0.9 {
		pip19Score = fmt.Sprintf("%v✅", nodeInfo.AvailabilityScore)
	} else {
		pip19Score = fmt.Sprintf("%v⚠️", nodeInfo.AvailabilityScore)
	}

	return cmd.SuccessfulResult("PeerID: %s\nIP Address: %s\nAgent: %s\n"+
		"Moniker: %s\nCountry: %s\nCity: %s\nRegion Name: %s\nTimeZone: %s\n"+
		"ISP: %s\n\nValidator Info🔍\nNumber: %v\nPIP-19 Score: %s\nStake: %v PAC's\n",
		nodeInfo.PeerID, nodeInfo.IPAddress, nodeInfo.Agent, nodeInfo.Moniker, nodeInfo.Country,
		nodeInfo.City, nodeInfo.RegionName, nodeInfo.TimeZone, nodeInfo.ISP, utils.FormatNumber(int64(nodeInfo.ValidatorNum)),
		pip19Score, utils.FormatNumber(nodeInfo.StakeAmount))
}
