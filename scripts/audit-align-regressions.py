"""Make full-form tests open the explicit advanced route, not the quick-flow redirect."""
from pathlib import Path
paths=['web/tests/product-flows.spec.ts','web/tests/accessibility.spec.ts','web/tests/audit-product-journeys.spec.ts','web/tests/audit-full-form.spec.ts']
prepared=[]
for name in paths:
 p=Path(name);s=p.read_text()
 assert "'/app/credit/new'" in s or "'/app/credit/new?organization=" in s,name
 s=s.replace("'/app/credit/new'","'/app/credit/new?advanced=1'").replace("'/app/credit/new?organization=","'/app/credit/new?advanced=1&organization=")
 prepared.append((p,s))
for p,s in prepared:p.write_text(s)
Path('.tmp').mkdir(exist_ok=True)
Path('.tmp/regression-paths.txt').write_text(''.join(path+'\n' for path in paths))
print('Full-form tests now target the explicit advanced route; assertions are unchanged.')
