package whatsapp

import "testing"

func TestPaymentCommandPreservesUnicodeBuyerNames(t *testing.T) {
	// Unicode lowercasing can change UTF-8 byte lengths. Delimiter offsets
	// must refer to the original message, not its lowercased copy.
	for _, name := range []string{"Kola", "İpek", "Ádé", "Chinedu"} {
		for _, verb := range []string{" paid ", " PAID ", " PaId "} {
			command, err := ParseCommand(name + verb + "123.45")
			if err != nil || command.Kind != CommandRecordPayment || command.BuyerName != name || command.AmountKobo != 12345 || !command.RequiresConfirmation {
				t.Fatalf("%q: got %+v, %v", name+verb, command, err)
			}
		}
	}
}
