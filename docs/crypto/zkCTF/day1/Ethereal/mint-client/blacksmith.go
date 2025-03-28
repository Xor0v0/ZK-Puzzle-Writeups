package main

import (
	"crypto"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/kzg"
	"github.com/davecgh/go-spew/spew"
)

func ForgeSword() []fr.Element {
	// initialize the fr slice
	f := make([]fr.Element, 16)
	// initialize the hash function
	h := crypto.SHA256.New()
	// get the current timestamp
	ts := time.Now().UnixNano()
	// convert the timestamp to bytes
	tsB := make([]byte, 8)
	binary.LittleEndian.PutUint64(tsB, uint64(ts))
	// write the timestamp to the hash function as input
	h.Write(tsB)
	h.Write([]byte("Soul of a Hero"))
	// get the hash of the input
	seed := h.Sum(nil)
	// iterate 10 times and fill the fr slice with the hash of the seed
	for i := 0; i < 16; i++ {
		f[i].SetBytes(seed[:32])
		h.Reset()
		h.Write(seed)
		seed = h.Sum(nil)
	}
	return f
}

// define the KeyPairProof struct
type KeyPairProof struct {
	H              bn254.G1Affine
	PrivateKey     *fr.Element
	PublicKeyG1Aff bn254.G1Affine
}

// CraftBladeSignature crafts a blade signature
func CraftBladeSignature(poly []fr.Element, srs *kzg.SRS) (kzg.Digest, *KeyPairProof) {
	//commit the polynomial
	commitment, err := kzg.Commit(poly, srs.Pk)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("commitment: \nX: %02x\nY: %02x\n", commitment.X, commitment.Y)

	// compute opening proof at a random point
	var point fr.Element
	point.SetInt64(0)
	proof, err := kzg.Open(poly, point, srs.Pk)
	if err != nil {
		panic(err)
	}

	// claimed value is private key.
	// derive public key from private key
	privateKey := new(big.Int)
	proof.ClaimedValue.BigInt(privateKey)

	publicKey := new(bn254.G1Affine)
	publicKey.ScalarMultiplication(&srs.Pk.G1[0], privateKey)
	spew.Dump(privateKey, publicKey)
	publicKeyG2 := new(bn254.G2Affine)
	publicKeyG2.ScalarMultiplication(&srs.Vk.G2[0], privateKey)

	pubKeyProof := new(KeyPairProof)
	pubKeyProof.PrivateKey = &proof.ClaimedValue
	pubKeyProof.H = proof.H
	pubKeyProof.PublicKeyG1Aff = *publicKey
	return commitment, pubKeyProof
}
