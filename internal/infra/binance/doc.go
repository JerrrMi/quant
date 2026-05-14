// Package binance implements USD-M Futures REST access for the Agent process.
// It maps HTTP errors to retry hints and exposes read-only market/account views plus order placement APIs.
// Binance does not use API passphrases; callers may still load a passphrase env for other venues elsewhere.
package binance
