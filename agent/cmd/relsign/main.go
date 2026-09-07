// relsign : génère une paire de clés ed25519 et signe/vérifie les fichiers de release (checksums.txt).
// Le CI signe avec la clé privée (secret GitHub RELEASE_SIGNING_KEY) ; l'agent vérifie avec la clé publique embarquée.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "keygen":
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		must(err)
		fmt.Printf("private=%s\n", base64.StdEncoding.EncodeToString(priv))
		fmt.Printf("public=%s\n", base64.StdEncoding.EncodeToString(pub))
	case "sign":
		fs := flag.NewFlagSet("sign", flag.ExitOnError)
		keyEnv := fs.String("key-env", "RELEASE_SIGNING_KEY", "variable d'environnement contenant la clé privée (base64)")
		out := fs.String("o", "", "fichier de signature (défaut : <fichier>.sig)")
		fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			usage()
		}
		raw, err := base64.StdEncoding.DecodeString(os.Getenv(*keyEnv))
		must(err)
		if len(raw) != ed25519.PrivateKeySize {
			fatal("clé privée invalide dans $" + *keyEnv)
		}
		data, err := os.ReadFile(fs.Arg(0))
		must(err)
		sig := ed25519.Sign(ed25519.PrivateKey(raw), data)
		if *out == "" {
			*out = fs.Arg(0) + ".sig"
		}
		must(os.WriteFile(*out, []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644))
		fmt.Println("signé :", *out)
	case "verify":
		fs := flag.NewFlagSet("verify", flag.ExitOnError)
		pubB64 := fs.String("pub", "", "clé publique (base64)")
		fs.Parse(os.Args[2:])
		if fs.NArg() != 2 || *pubB64 == "" {
			usage()
		}
		pub, err := base64.StdEncoding.DecodeString(*pubB64)
		must(err)
		data, err := os.ReadFile(fs.Arg(0))
		must(err)
		sigB64, err := os.ReadFile(fs.Arg(1))
		must(err)
		sig, err := base64.StdEncoding.DecodeString(string(trim(sigB64)))
		must(err)
		if !ed25519.Verify(ed25519.PublicKey(pub), data, sig) {
			fatal("signature INVALIDE")
		}
		fmt.Println("signature valide")
	default:
		usage()
	}
}

func trim(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage : relsign keygen | relsign sign [-key-env VAR] [-o out] <fichier> | relsign verify -pub <base64> <fichier> <fichier.sig>")
	os.Exit(2)
}
func must(err error) {
	if err != nil {
		fatal(err.Error())
	}
}
func fatal(msg string) { fmt.Fprintln(os.Stderr, "erreur :", msg); os.Exit(1) }
