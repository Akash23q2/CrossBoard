// imports
package main

import (
	"os"

	"github.com/mdp/qrterminal/v3"
)

func generateQr(content string) {
	qrterminal.Generate(content, qrterminal.M, os.Stdout)
}
