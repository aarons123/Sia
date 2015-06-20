package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/NebulousLabs/Sia/api"
	"github.com/NebulousLabs/Sia/modules"
)

// coinUnits converts a siacoin amount to base units.
func coinUnits(amount string) (string, error) {
	units := []string{"pS", "nS", "uS", "mS", "SC", "KS", "MS", "GS", "TS"}
	for i, unit := range units {
		if strings.HasSuffix(amount, unit) {
			base := strings.TrimSuffix(amount, unit)
			zeros := 27 + 3*(i-3)
			// may need to adjust for non-integer values
			if d := strings.IndexByte(base, '.'); d != -1 {
				zeros -= len(base) - d - 1
				if zeros < 0 {
					return "", errors.New("non-integer number of hastings")
				}
				base = base[:d] + base[d+1:]
			}
			return base + strings.Repeat("0", zeros), nil
		}
	}
	return amount, nil // hastings
}

var (
	walletCmd = &cobra.Command{
		Use:   "wallet",
		Short: "Perform wallet actions",
		Long:  "Generate a new address, send coins to another wallet, or view info about the wallet.",
		Run:   wrap(walletstatuscmd),
	}

	walletAddressCmd = &cobra.Command{
		Use:   "address",
		Short: "Get a new wallet address",
		Long:  "Generate a new wallet address.",
		Run:   wrap(walletaddresscmd),
	}

	walletSendCmd = &cobra.Command{
		Use:   "send [amount] [dest]",
		Short: "Send coins to another wallet",
		Long: `Send coins to another wallet. 'dest' must be a 64-byte hexadecimal address.
'amount' can be specified in units, e.g. 1.23KS. Supported units are:
	pS (pico,  10^-12 SC)
	nS (nano,  10^-9 SC)
	uS (micro, 10^-6 SC)
	mS (milli, 10^-3 SC)
	SC
	KS (kilo, 10^3 SC)
	MS (mega, 10^6 SC)
	GS (giga, 10^9 SC)
	TS (tera, 10^12 SC)
If no unit is supplied, hastings (smallest possible unit, 10^-27 SC) will be assumed.
`,
		Run: wrap(walletsendcmd),
	}

	walletSiafundsCmd = &cobra.Command{
		Use:   "siafunds",
		Short: "Display siafunds balance",
		Long:  "Display siafunds balance and siacoin claim balance.",
		Run:   wrap(walletsiafundscmd),
	}

	walletSiafundsSendCmd = &cobra.Command{
		Use:   "send [amount] [dest] [keyfiles]",
		Short: "Send siafunds",
		Long: `Send siafunds to an address, and transfer their siacoins to the wallet.
Run 'wallet send --help' to see a list of available units.`,
		Run: walletsiafundssendcmd, // see function docstring
	}

	walletSiafundsTrackCmd = &cobra.Command{
		Use:   "track [keyfile]",
		Short: "Track a siafund address generated by siag",
		Long:  "Track a siafund address generated by siag.",
		Run:   wrap(walletsiafundstrackcmd),
	}

	walletStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "View wallet status",
		Long:  "View wallet status, including the current balance and number of addresses.",
		Run:   wrap(walletstatuscmd),
	}
)

// TODO: this should be defined outside of siac
type walletAddr struct {
	Address string
}

func walletaddresscmd() {
	addr := new(walletAddr)
	err := getAPI("/wallet/address", addr)
	if err != nil {
		fmt.Println("Could not generate new address:", err)
		return
	}
	fmt.Printf("Created new address: %s\n", addr.Address)
}

func walletsendcmd(amount, dest string) {
	adjAmount, err := coinUnits(amount)
	if err != nil {
		fmt.Println("Could not parse amount:", err)
		return
	}
	err = post("/wallet/send", fmt.Sprintf("amount=%s&destination=%s", adjAmount, dest))
	if err != nil {
		fmt.Println("Could not send:", err)
		return
	}
	fmt.Printf("Sent %s to %s\n", adjAmount, dest)
}

func walletsiafundscmd() {
	bal := new(api.WalletSiafundsBalance)
	err := getAPI("/wallet/siafunds/balance", bal)
	if err != nil {
		fmt.Println("Could not get siafunds balance:", err)
		return
	}
	fmt.Printf("Siafunds Balance: %s\nClaim Balance: %s\n", bal.SiafundBalance, bal.SiacoinClaimBalance)
}

// special because list of keyfiles is variadic
func walletsiafundssendcmd(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		cmd.Usage()
		return
	}
	amount, err := coinUnits(args[0])
	if err != nil {
		fmt.Println("Could not parse amount:", err)
		return
	}
	dest, keyfiles := args[1], args[2:]
	for i := range keyfiles {
		keyfiles[i] = abs(keyfiles[i])
	}

	qs := fmt.Sprintf("amount=%s&destination=%s&keyfiles=%s", amount, dest, strings.Join(keyfiles, ","))

	err = post("/wallet/siafunds/send", qs)
	if err != nil {
		fmt.Println("Could not track siafunds:", err)
		return
	}
	fmt.Printf("Sent %s siafunds to %s\n", amount, dest)
}

func walletsiafundstrackcmd(keyfile string) {
	err := post("/wallet/siafunds/watchsiagaddress", "keyfile="+abs(keyfile))
	if err != nil {
		fmt.Println("Could not track siafunds:", err)
		return
	}
	fmt.Printf(`Added %s to tracked siafunds.

You must restart siad to update your siafund balance.
Do not delete the original keyfile.
`, keyfile)
}

func walletstatuscmd() {
	status := new(modules.WalletInfo)
	err := getAPI("/wallet/status", status)
	if err != nil {
		fmt.Println("Could not get wallet status:", err)
		return
	}
	fmt.Printf(`Wallet status:
Balance:   %v (confirmed)
           %v (unconfirmed)
Addresses: %d
`, status.Balance, status.FullBalance, status.NumAddresses)
}
