// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ed25519

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"testing"
)

type zeroReader struct{}

func (zeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func TestSignVerify(t *testing.T) {
	var zero zeroReader
	public, private, _ := GenerateKey(zero, sha512.New())

	message := []byte("test message")
	sig := Sign(private, message, sha512.New())
	if !Verify(public, message, sig, sha512.New()) {
		t.Errorf("valid signature rejected")
	}

	wrongMessage := []byte("wrong message")
	if Verify(public, wrongMessage, sig, sha512.New()) {
		t.Errorf("signature of different message accepted")
	}
}

func TestGolden(t *testing.T) {
	// sign.input.gz is a selection of test cases from
	// http://ed25519.cr.yp.to/python/sign.input
	testDataZ, err := os.Open("testdata/sign.input.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer testDataZ.Close()
	testData, err := gzip.NewReader(testDataZ)
	if err != nil {
		t.Fatal(err)
	}
	defer testData.Close()

	in := bufio.NewReaderSize(testData, 1<<12)
	lineNo := 0
	for {
		lineNo++
		lineBytes, isPrefix, err := in.ReadLine()
		if isPrefix {
			t.Fatal("bufio buffer too small")
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("error reading test data: %s", err)
		}

		line := string(lineBytes)
		parts := strings.Split(line, ":")
		if len(parts) != 5 {
			t.Fatalf("bad number of parts on line %d", lineNo)
		}

		privBytes, _ := hex.DecodeString(parts[0])
		pubKeyBytes, _ := hex.DecodeString(parts[1])
		msg, _ := hex.DecodeString(parts[2])
		sig, _ := hex.DecodeString(parts[3])
		// The signatures in the test vectors also include the message
		// at the end, but we just want R and S.
		sig = sig[:SignatureSize]

		if l := len(pubKeyBytes); l != PublicKeySize {
			t.Fatalf("bad public key length on line %d: got %d bytes", lineNo, l)
		}

		var priv [PrivateKeySize]byte
		copy(priv[:], privBytes)
		copy(priv[32:], pubKeyBytes)

		sig2 := Sign(PrivateKey(priv[:]), msg, sha512.New())
		if !bytes.Equal(sig, sig2[:]) {
			t.Errorf("different signature result on line %d: %x vs %x", lineNo, sig, sig2)
		}

		var pubKey [PublicKeySize]byte
		copy(pubKey[:], pubKeyBytes)
		if !Verify(PublicKey(pubKey[:]), msg, sig2, sha512.New()) {
			t.Errorf("signature failed to verify on line %d", lineNo)
		}
	}
}

func BenchmarkKeyGeneration(b *testing.B) {
	var zero zeroReader
	for i := 0; i < b.N; i++ {
		if _, _, err := GenerateKey(zero, sha512.New()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigning(b *testing.B) {
	var zero zeroReader
	_, priv, err := GenerateKey(zero, sha512.New())
	if err != nil {
		b.Fatal(err)
	}
	message := []byte("Hello, world!")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(priv, message, sha512.New())
	}
}

func BenchmarkVerification(b *testing.B) {
	var zero zeroReader
	pub, priv, err := GenerateKey(zero, sha512.New())
	if err != nil {
		b.Fatal(err)
	}
	message := []byte("Hello, world!")
	signature := Sign(priv, message, sha512.New())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(pub, message, signature, sha512.New())
	}
}
