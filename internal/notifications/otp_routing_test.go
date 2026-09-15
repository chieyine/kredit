package notifications

import (
	"context"
	"testing"
)

func TestAuthenticationDeliveryChannels(t *testing.T) {
	ctx := context.Background()
	s := NewStore("otp-routing-test")
	email, whatsapp, sms := NewMockProvider(ChannelEmail), NewMockProvider(ChannelWhatsApp), NewMockProvider(ChannelSMS)
	s.RegisterProvider(email)
	s.RegisterProvider(whatsapp)
	s.RegisterProvider(sms)
	if err := s.SendOTP(ctx, "owner@example.test", "email", "123456"); err != nil {
		t.Fatal(err)
	}
	if err := s.SendOTP(ctx, "+2348012345678", "phone", "654321"); err != nil {
		t.Fatal(err)
	}
	if len(email.Messages()) != 1 || len(whatsapp.Messages()) != 1 || len(sms.Messages()) != 0 {
		t.Fatal("email and phone codes must use email and WhatsApp respectively")
	}
	m := whatsapp.Messages()[0]
	if m.Template != "AuthenticationCode" || m.AuthenticationCode != "654321" || m.Destination != "+2348012345678" {
		t.Fatal("WhatsApp authentication template must receive the code and exact phone destination")
	}
	whatsapp.SetFail(true)
	if err := s.SendOTP(ctx, "+2348012345678", "phone", "654321"); err == nil {
		t.Fatal("WhatsApp failure must be reported")
	}
	if len(sms.Messages()) != 0 || len(email.Messages()) != 1 {
		t.Fatal("failed phone authentication must not silently switch channels")
	}
	whatsapp.SetFail(false)
	if err := s.SendRecoveryInstructions(ctx, "+2348012345678", "phone", "https://kredit.ng/recover"); err != nil {
		t.Fatal(err)
	}
	if len(whatsapp.Messages()) != 2 || len(sms.Messages()) != 0 {
		t.Fatal("phone recovery instructions must use WhatsApp")
	}
}
