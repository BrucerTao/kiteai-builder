#!/usr/bin/env python3
# Decode a base64 x402 PAYMENT-REQUIRED header and assert Kite testnet params.
# Usage: decode_payment_required.py [b64-file]   (or pipe the b64 on stdin). Masks 0x addresses.
import base64, json, sys


def load():
    raw = open(sys.argv[1]).read().strip() if len(sys.argv) > 1 else sys.stdin.read().strip()
    if ':' in raw:
        head = raw.split(':')[0].lower().replace('-', '').replace('_', '')
        if head in ('xpaymentrequired', 'paymentrequired'):
            raw = raw.split(':', 1)[1].strip()
    pad = raw + '=' * (-len(raw) % 4)
    try:
        return json.loads(base64.b64decode(pad))
    except Exception:
        return json.loads(base64.urlsafe_b64decode(pad))


def mask(o):
    if isinstance(o, dict):
        return {k: mask(v) for k, v in o.items()}
    if isinstance(o, list):
        return [mask(v) for v in o]
    if isinstance(o, str) and o.startswith('0x') and len(o) >= 12:
        return o[:6] + '...' + o[-4:]
    return o


def main():
    d = load()
    print('=== decoded PAYMENT-REQUIRED (0x addresses masked) ===')
    print(json.dumps(mask(d), indent=2, ensure_ascii=False))
    acc = (d.get('accepts') or [{}])[0]
    extra = acc.get('extra') or {}
    amount = acc.get('maxAmountRequired', acc.get('amount'))
    checks = [
        ('accepts[0].network == eip155:2368', acc.get('network') == 'eip155:2368'),
        ('extra.name == pieUSD', extra.get('name') == 'pieUSD'),
        ('extra.version == 1', str(extra.get('version')) == '1'),
        ('amount == 1000000000000000', str(amount) == '1000000000000000'),
    ]
    print('=== assertions (Kite testnet) ===')
    for label, ok in checks:
        print(('PASS' if ok else 'FAIL'), '-', label)
    ok_all = all(ok for _, ok in checks)
    print('RESULT:', 'ALL_PASS' if ok_all else 'SOME_FAIL')
    sys.exit(0 if ok_all else 1)


if __name__ == '__main__':
    main()
