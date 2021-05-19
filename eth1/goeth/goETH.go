package goeth

import (
	"context"
	"encoding/hex"
	"github.com/bloxapp/ssv/storage/collections"
	"github.com/bloxapp/ssv/utils/rsaencryption"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/bloxapp/ssv/eth1"
	"github.com/bloxapp/ssv/shared/params"
)

type eth1GRPC struct {
	ctx             context.Context
	conn            *ethclient.Client
	logger          *zap.Logger
	contractEvent   *eth1.ContractEvent
	operatorStorage collections.IOperatorStorage
}

// New create new goEth instance
func New(ctx context.Context, logger *zap.Logger, nodeAddr string, operatorStorage collections.IOperatorStorage) (eth1.Eth1, error) {
	// Create an IPC based RPC connection to a remote node
	conn, err := ethclient.Dial(nodeAddr)
	if err != nil {
		logger.Error("Failed to connect to the Ethereum client", zap.Error(err))
	}

	e := &eth1GRPC{
		ctx:             ctx,
		conn:            conn,
		logger:          logger,
		operatorStorage: operatorStorage,
	}

	// init the instance which publishes an event when anything happens
	err = e.streamSmartContractEvents(params.SsvConfig().OperatorContractAddress)
	if err != nil {
		logger.Error("Failed to init operator contract address subject", zap.Error(err))
	}

	return e, nil
}

// streamSmartContractEvents implements Eth1 interface
func (e *eth1GRPC) streamSmartContractEvents(contractAddr string) error {
	contractAddress := common.HexToAddress(contractAddr)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}

	logs := make(chan types.Log)
	sub, err := e.conn.SubscribeFilterLogs(e.ctx, query, logs)
	if err != nil {
		e.logger.Fatal("Failed to subscribe to logs", zap.Error(err))
		return err
	}

	contractAbi, err := abi.JSON(strings.NewReader(params.SsvConfig().ContractABI))
	if err != nil {
		e.logger.Fatal("Failed to parse ABI interface", zap.Error(err))
	}

	e.contractEvent = eth1.NewContractEvent("smartContractEvent")
	go func() {
		for {
			select {
			case err := <-sub.Err():
				// TODO might fail consider reconnect
				e.logger.Error("Error from logs sub", zap.Error(err))

			case vLog := <-logs:
				eventType, err := contractAbi.EventByID(vLog.Topics[0])
				if err != nil {
					e.logger.Error("Failed to get event by topic hash", zap.Error(err))
					continue
				}

				switch eventName := eventType.Name; eventName {
				case "OperatorAdded":
					operatorAddedEvent := eth1.OperatorAddedEvent{}
					err = contractAbi.UnpackIntoInterface(&operatorAddedEvent, eventType.Name, vLog.Data)
					if err != nil {
						e.logger.Error("Failed to unpack event", zap.Error(err))
						continue
					}
					e.contractEvent.Data = operatorAddedEvent

				case "ValidatorAdded":
					validatorAddedEvent := eth1.ValidatorAddedEvent{}
					err = contractAbi.UnpackIntoInterface(&validatorAddedEvent, eventType.Name, vLog.Data)
					if err != nil {
						e.logger.Error("Failed to unpack ValidatorAdded event", zap.Error(err))
						continue
					}

					isEventBelongsToOperator := false

					e.logger.Debug("ValidatorAdded Event",
						zap.String("Validator PublicKey", hex.EncodeToString(validatorAddedEvent.PublicKey)),
						zap.String("Owner Address", validatorAddedEvent.OwnerAddress.String()))
					for i := range validatorAddedEvent.OessList {
						validatorShare := validatorAddedEvent.OessList[i]
						e.logger.Debug("Validator Share",
							zap.Any("Index", validatorShare.Index),
							zap.String("Operator PubKey", hex.EncodeToString(validatorShare.OperatorPublicKey)),
							zap.String("Share PubKey", hex.EncodeToString(validatorShare.SharedPublicKey)),
							zap.String("Encrypted Key", hex.EncodeToString(validatorShare.EncryptedKey)))

						if strings.EqualFold(hex.EncodeToString(validatorShare.OperatorPublicKey), params.SsvConfig().OperatorPublicKey) {
							sk, err := e.operatorStorage.GetPrivateKey()
							if err != nil{
								e.logger.Error("failed to get private key", zap.Error(err))
								continue
							}
							decryptedShare, err := rsaencryption.DecodeKey(sk, string(validatorShare.EncryptedKey))
							if err != nil{
								e.logger.Error("failed to decrypt share key", zap.Error(err))
								continue
							}
							validatorShare.EncryptedKey = []byte(decryptedShare)

							isEventBelongsToOperator = true
						}
					}

					if isEventBelongsToOperator {
						e.contractEvent.Data = validatorAddedEvent
						e.contractEvent.NotifyAll()
					}

				default:
					e.logger.Debug("Unknown contract event is received")
					continue
				}
			}
		}
	}()
	return nil
}

func (e *eth1GRPC) GetContractEvent() *eth1.ContractEvent {
	return e.contractEvent
}

// deleteEmpty TODO need this func?
//func deleteEmpty(s []string) []string {
//	var r []string
//	for _, str := range s {
//		if str != "" {
//			r = append(r, str)
//		}
//	}
//	return r
//}
