#!/usr/bin/env python3
from pathlib import Path

root = Path(__file__).resolve().parents[1]

buyer = root / 'internal/web/buyer_handlers.go'
text = buyer.read_text(encoding='utf-8')
text = text.replace('\t"context"\n', '')
text = text.replace('s.runtime.Buyers.Accept(context.Background(), token, user.ID,', 's.runtime.Buyers.Accept(r.Context(), token, user.ID,')
buyer.write_text(text, encoding='utf-8')

onboarding = root / 'internal/web/onboarding_handlers.go'
text = onboarding.read_text(encoding='utf-8')
text = text.replace('\t"context"\n', '')
text = text.replace('s.runtime.EmitNotification(context.Background(),', 's.runtime.EmitNotification(r.Context(),')
onboarding.write_text(text, encoding='utf-8')
