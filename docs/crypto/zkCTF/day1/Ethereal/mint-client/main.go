package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"zk/mint/mint"
	"zk/mint/utils"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/kzg"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	SRS kzg.SRS
)

const (
	SRS_SIZE = 16
)

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func loadSRSFromContract(mintContract *mint.MintCaller) *kzg.SRS {
	SRS := kzg.SRS{
		Pk: kzg.ProvingKey{
			G1: make([]bn254.G1Affine, SRS_SIZE),
		},
		Vk: kzg.VerifyingKey{
			G2: [2]bn254.G2Affine{},
			G1: bn254.G1Affine{},
		},
	}
	for i := 0; i < SRS_SIZE; i++ {
		g1_x, err := mintContract.SRSG1X(nil, big.NewInt(int64(i)))
		panicErr(err)
		SRS.Pk.G1[i].X.SetBigInt(g1_x)
		g1_y, err := mintContract.SRSG1Y(nil, big.NewInt(int64(i)))
		panicErr(err)
		SRS.Pk.G1[i].Y.SetBigInt(g1_y)
	}
	SRS.Vk.G1.X.Set(&SRS.Pk.G1[0].X)
	SRS.Vk.G1.Y.Set(&SRS.Pk.G1[0].Y)

	zero := big.NewInt(0)
	one := big.NewInt(1)
	g2_x0_0, err := mintContract.SRSG2X0(nil, zero)
	panicErr(err)
	SRS.Vk.G2[0].X.A0.SetBigInt(g2_x0_0)
	g2_x0_1, err := mintContract.SRSG2X0(nil, one)
	panicErr(err)
	SRS.Vk.G2[0].X.A1.SetBigInt(g2_x0_1)

	g2_x1_0, err := mintContract.SRSG2X1(nil, zero)
	panicErr(err)
	SRS.Vk.G2[1].X.A0.SetBigInt(g2_x1_0)
	g2_x1_1, err := mintContract.SRSG2X1(nil, one)
	panicErr(err)
	SRS.Vk.G2[1].X.A1.SetBigInt(g2_x1_1)

	g2_y0_0, err := mintContract.SRSG2Y0(nil, zero)
	panicErr(err)
	SRS.Vk.G2[0].Y.A0.SetBigInt(g2_y0_0)
	g2_y0_1, err := mintContract.SRSG2Y0(nil, one)
	panicErr(err)
	SRS.Vk.G2[0].Y.A1.SetBigInt(g2_y0_1)

	g2_y1_0, err := mintContract.SRSG2Y1(nil, zero)
	panicErr(err)
	SRS.Vk.G2[1].Y.A0.SetBigInt(g2_y1_0)
	g2_y1_1, err := mintContract.SRSG2Y1(nil, one)
	panicErr(err)
	SRS.Vk.G2[1].Y.A1.SetBigInt(g2_y1_1)
	return &SRS
}

func G1AffineToG1Point(p *bn254.G1Affine) *mint.PairingG1Point {
	var X, Y big.Int
	p.X.BigInt(&X)
	p.Y.BigInt(&Y)
	return &mint.PairingG1Point{
		X: &X,
		Y: &Y,
	}
}

func ordinal(n int) string {
	suffix := []string{"th", "st", "nd", "rd"}
	if n%100 >= 11 && n%100 <= 13 {
		return strconv.Itoa(n) + "th"
	}
	return strconv.Itoa(n) + suffix[n%10%4]
}

func main() {
	var action = flag.String("action", "register", "action to perform")
	var contractAddr = flag.String("contract", "", "contract address")
	var rpcAddr = flag.String("rpc", "http://127.0.0.1:8545", "rpc address")
	// --------------------------------------------------
	// Note: Please change the account and pk to your own
	var accountAddr = flag.String("account", "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "account address")
	var privateKey = flag.String("pk", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", "private key")
	// --------------------------------------------------
	flag.Parse()
	contractAddress := common.HexToAddress(*contractAddr)
	accountAddress := common.HexToAddress(*accountAddr)
	ethclient, err := ethclient.Dial(*rpcAddr)
	transactor := utils.NewTransactor(ethclient, *privateKey)

	mintContract, err := mint.NewMint(contractAddress, ethclient)
	panicErr(err)

	SRS := loadSRSFromContract(&mintContract.MintCaller)

	switch *action {
	case "register":
		sword := ForgeSword()
		bladeCommitment, bladeProof := CraftBladeSignature(sword, SRS)
		commitmentPoint := G1AffineToG1Point(&bladeCommitment)
		bladeProofPoint := G1AffineToG1Point(&bladeProof.H)
		soulBox := G1AffineToG1Point(&bladeProof.PublicKeyG1Aff)
		mintContract.Register(transactor, *commitmentPoint, *bladeProofPoint, *soulBox)
		f, err := os.OpenFile("sword.json", os.O_CREATE|os.O_WRONLY, 0644)
		panicErr(err)
		defer f.Close()
		f.Truncate(0)
		f.Seek(0, 0)
		encoder := json.NewEncoder(f)
		encoder.SetIndent("", "  ")
		encoder.Encode(sword)
		transactor.Value = big.NewInt(0)
	case "replay":
		mintContract.Replay(transactor)
	case "mint":
		var sword []fr.Element
		f, err := os.Open("sword.json")
		panicErr(err)
		defer f.Close()
		decoder := json.NewDecoder(f)
		err = decoder.Decode(&sword)
		panicErr(err)

		nonce, err := mintContract.GetNonce(nil, transactor.From)
		panicErr(err)
		nonceFr := new(fr.Element)
		nonceFr.SetBigInt(nonce)
		fmt.Printf("Minting the %s gem\n", ordinal(int(nonce.Int64())))
		proof, err := kzg.Open(sword, *nonceFr, SRS.Pk)
		panicErr(err)
		proofPoint := G1AffineToG1Point(&proof.H)

		var value big.Int
		proof.ClaimedValue.BigInt(&value)
		mintContract.Mint(transactor, *proofPoint, &value)
	case "query":
		accountDeposit, err := mintContract.Deposits(nil, accountAddress)
		panicErr(err)
		collectedY, err := mintContract.GetCollectedY(nil, accountAddress)
		panicErr(err)
		spew.Dump(collectedY, accountDeposit)
	case "forge":
		// 1. load sword info
		var sword []fr.Element
		f, err := os.Open("sword.json")
		panicErr(err)
		defer f.Close()
		decoder := json.NewDecoder(f)
		err = decoder.Decode(&sword)
		panicErr(err)

		// 2. forge
		bladeCommitment, bladeProof := CraftBladeSignature(sword, SRS)
		commitmentPoint := G1AffineToG1Point(&bladeCommitment)
		// forge the fake bladeProof
		var fakeProof bn254.G1Affine
		fakeProof.ScalarMultiplication(&SRS.Pk.G1[0], big.NewInt(114514)) // g^{114514})
		spew.Dump(fakeProof)
		fakeProof.Add(&fakeProof, &bladeProof.H)
		spew.Dump(fakeProof)
		bladeProofPoint := G1AffineToG1Point(&fakeProof)
		spew.Dump(bladeProofPoint)

		var fakeSoulBox bn254.G1Affine
		fakeSoulBox.ScalarMultiplication(&SRS.Pk.G1[1], big.NewInt(114514)) // g^{\tau * 114514})
		fakeSoulBox.Sub(&bladeProof.PublicKeyG1Aff, &fakeSoulBox)
		soulBox := G1AffineToG1Point(&fakeSoulBox)

		// 3. regiter
		mintContract.Register(transactor, *commitmentPoint, *bladeProofPoint, *soulBox)
	}

}
