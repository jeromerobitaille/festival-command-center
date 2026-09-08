#!/bin/sh
# Régénère hub/dashboard/data/oui.tsv.gz depuis le registre OUI de l'IEEE.
# À relancer de temps en temps : le registre gagne quelques centaines d'entrées par an.
set -e
cd "$(dirname "$0")/.."
curl -sSf --max-time 120 -o /tmp/oui.csv https://standards-oui.ieee.org/oui/oui.csv
python3 - <<'PY'
import csv, gzip, io, re, os
rows = []
with open('/tmp/oui.csv', newline='', encoding='utf-8') as f:
    for r in csv.DictReader(f):
        oui = (r.get('Assignment') or '').strip().lower()
        org = re.sub(r'\s+', ' ', (r.get('Organization Name') or '').strip())
        if len(oui) != 6 or not re.fullmatch(r'[0-9a-f]{6}', oui) or not org:
            continue
        org = re.sub(r'[,\s]+(Inc|Ltd|LLC|Corp|Corporation|Co|GmbH|S\.A|B\.V|A/S|Pte|Pty|SAS|SARL|AB|AG|NV|SpA|Limited|Technologies|Technology)\.?$', '', org, flags=re.I).strip(' .,')
        if org:
            rows.append((oui, org[:44]))
rows.sort()
buf = io.StringIO()
for o, n in rows:
    buf.write(o + '\t' + n + '\n')
with gzip.open('data/oui.tsv.gz', 'wb', compresslevel=9) as g:
    g.write(buf.getvalue().encode('utf-8'))
print(f"{len(rows)} entrees -> data/oui.tsv.gz ({os.path.getsize('data/oui.tsv.gz')//1024} Ko)")
PY
