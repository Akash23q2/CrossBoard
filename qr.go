// imports
package main

import (
	"os"

	"github.com/mdp/qrterminal/v3"
)

func generateQr(content string) {
	qrterminal.GenerateHalfBlock(content, qrterminal.L, os.Stdout)
}
