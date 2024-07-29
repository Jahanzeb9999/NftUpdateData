package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"


	"github.com/CoreumFoundation/coreum/v4/pkg/client"
	coreumconfig "github.com/CoreumFoundation/coreum/v4/pkg/config"
	"github.com/CoreumFoundation/coreum/v4/pkg/config/constant"
	assetnfttypes "github.com/CoreumFoundation/coreum/v4/x/asset/nft/types"
)

const (
	// Replace it with your own mnemonic
	senderMnemonic = "hair album dose tribe vendor risk inmate helmet size artefact sadness repeat laugh range access this target picture develop parent quarter trap either very"

	chainID       = constant.ChainIDDev
	addressPrefix = constant.AddressPrefixDev
	nodeAddress   = "full-node.devnet-1.coreum.dev:9090"
)

type Response struct {
	Message string `json:"message"`
}

type CreateNFTClassRequest struct {
	ClassSymbol      string `json:"classSymbol"`
	ClassName        string `json:"className"`
	ClassDescription string `json:"classDescription"`
}

type MintNFTRequest struct {
	ClassSymbol string `json:"classSymbol"`
	NFTID       string `json:"nftID"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateNFTDataRequest struct {
	ClassID     string `json:"classID"`
	NFTID       string `json:"nftID"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func main() {
	// Configure Cosmos SDK
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(addressPrefix, addressPrefix+"pub")
	config.SetCoinType(constant.CoinType)
	config.Seal()

	// Initialize router
	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		response := Response{Message: "Hello from Go!"}
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	r.HandleFunc("/api/create-class", createNFTClassHandler).Methods("POST")
	r.HandleFunc("/api/mint", mintNFTHandler).Methods("POST")
	r.HandleFunc("/api/update", updateNFTDataHandler).Methods("POST")

	// Serve static files
	staticDir := "./frontend/build"
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))

	// Run the server
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func createNFTClassHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateNFTClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	clientCtx, txFactory, senderAddress, err := setupClientContext()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msgIssueClass := &assetnfttypes.MsgIssueClass{
		Issuer:      senderAddress.String(),
		Symbol:      req.ClassSymbol,
		Name:        req.ClassName,
		Description: req.ClassDescription,
		Features:    []assetnfttypes.ClassFeature{assetnfttypes.ClassFeature_freezing},
	}

	_, err = client.BroadcastTx(ctx, clientCtx.WithFromAddress(senderAddress), txFactory, msgIssueClass)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(Response{Message: "NFT class created successfully"})
}

func mintNFTHandler(w http.ResponseWriter, r *http.Request) {
	var req MintNFTRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	clientCtx, txFactory, senderAddress, err := setupClientContext()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData := []byte(fmt.Sprintf(`{"name": "%s", "description": "%s"}`, req.Name, req.Description))
	dataDynamic := assetnfttypes.DataDynamic{
		Items: []assetnfttypes.DataDynamicItem{
			{
				Editors: []assetnfttypes.DataEditor{assetnfttypes.DataEditor_owner},
				Data:    jsonData,
			},
		},
	}
	anyData, err := codectypes.NewAnyWithValue(&dataDynamic)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	classID := assetnfttypes.BuildClassID(req.ClassSymbol, senderAddress)
	msgMint := &assetnfttypes.MsgMint{
		Sender:  senderAddress.String(),
		ClassID: classID,
		ID:      req.NFTID,
		Data:    anyData,
	}

	_, err = client.BroadcastTx(ctx, clientCtx.WithFromAddress(senderAddress), txFactory, msgMint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(Response{Message: "NFT minted successfully"})
}

func updateNFTDataHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateNFTDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	clientCtx, txFactory, senderAddress, err := setupClientContext()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData := []byte(fmt.Sprintf(`{"name": "%s", "description": "%s"}`, req.Name, req.Description))
	dataDynamic := assetnfttypes.DataDynamic{
		Items: []assetnfttypes.DataDynamicItem{
			{
				Editors: []assetnfttypes.DataEditor{assetnfttypes.DataEditor_owner},
				Data:    jsonData,
			},
		},
	}
	_, err = codectypes.NewAnyWithValue(&dataDynamic)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msgUpdateData := &assetnfttypes.MsgUpdateData{
		Sender:  senderAddress.String(),
		ClassID: req.ClassID,
		ID:      req.NFTID,
		Items: []assetnfttypes.DataDynamicIndexedItem{
			{
				Index: 0,
				Data:  jsonData,
			},
		},
	}

	_, err = client.BroadcastTx(ctx, clientCtx.WithFromAddress(senderAddress), txFactory, msgUpdateData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(Response{Message: "NFT data updated successfully"})
}

func setupClientContext() (client.Context, client.Factory, sdk.AccAddress, error) {
	// List required modules.
	modules := module.NewBasicManager(
		auth.AppModuleBasic{},
	)

	grpcClient, err := grpc.Dial(
		nodeAddress,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})),
	)
	if err != nil {
		return client.Context{}, client.Factory{}, nil, err
	}

	encodingConfig := coreumconfig.NewEncodingConfig(modules)

	clientCtx := client.NewContext(client.DefaultContextConfig(), modules).
		WithChainID(string(chainID)).
		WithGRPCClient(grpcClient).
		WithKeyring(keyring.NewInMemory(encodingConfig.Codec)).
		WithBroadcastMode(flags.BroadcastSync).
		WithAwaitTx(true)

	txFactory := client.Factory{}.
		WithKeybase(clientCtx.Keyring()).
		WithChainID(clientCtx.ChainID()).
		WithTxConfig(clientCtx.TxConfig()).
		WithSimulateAndExecute(true)

	senderInfo, err := clientCtx.Keyring().NewAccount(
		"key-name",
		senderMnemonic,
		"",
		sdk.GetConfig().GetFullBIP44Path(),
		hd.Secp256k1,
	)
	if err != nil {
		return client.Context{}, client.Factory{}, nil, err
	}

	senderAddress, err := senderInfo.GetAddress()
	if err != nil {
		return client.Context{}, client.Factory{}, nil, err
	}

	return clientCtx, txFactory, senderAddress, nil
}
