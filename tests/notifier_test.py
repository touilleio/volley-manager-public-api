"""Local consumer regression tests; AWS, WhatsApp and Slack are all mocked."""
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts/sqs-whatsapp-notifier.sh"


class NotifierTests(unittest.TestCase):
    def run_consumer(self, bodies, fail_send=0, restart=False):
        with tempfile.TemporaryDirectory() as directory:
            work = Path(directory)
            batches = [[{"Body": body, "MessageId": f"message-{i}", "ReceiptHandle": f"receipt-{i}"}]
                       for i, body in enumerate(bodies)]
            (work / "batches.json").write_text(json.dumps(batches))
            mocks = {
                "aws": '''
import json, os, signal, sys
from pathlib import Path
p = Path(os.environ['TEST_WORK'])
if sys.argv[2] == 'receive-message':
    n = int((p/'received').read_text()) if (p/'received').exists() else 0
    batches = json.loads((p/'batches.json').read_text())
    if n == int(os.environ['STOP_AFTER']):
        os.killpg(os.getpgrp(), signal.SIGTERM)
    (p/'received').write_text(str(n+1))
    print(json.dumps({'Messages': batches[n]}))
elif sys.argv[2] == 'delete-message':
    with (p/'deleted').open('a') as f: f.write(sys.argv[-1]+'\\n')
''',
                "wacli": '''
import os, sys
from pathlib import Path
p = Path(os.environ['TEST_WORK'])
with (p/'sent').open('a') as f: f.write(sys.argv[sys.argv.index('--message')+1].replace('\\n',' ')+'\\n')
n = len((p/'sent').read_text().splitlines())
if n == int(os.environ['FAIL_SEND']): sys.exit(1)
''',
                "curl": '''
import os, sys
from pathlib import Path
p = Path(os.environ['TEST_WORK'])
with (p/'slack').open('a') as f: f.write(sys.argv[sys.argv.index('--data')+1]+'\\n')
''',
            }
            for name, code in mocks.items():
                executable = work / name
                executable.write_text(f"#!{sys.executable}\n" + code)
                executable.chmod(0o755)
            env = os.environ.copy()
            env.pop("AWS_BIN", None)  # Exercise the documented PATH-based default.
            env.update(PATH=f"{work}:{env['PATH']}", WACLI_BIN="wacli", QUEUE_URL="mock-queue",
                       WACLI_TO="mock-recipient", SLACK_WEBHOOK_URL="https://mock.invalid/webhook",
                       STATE_DIR=str(work / "receipts"), TMPDIR=str(work), TEST_WORK=str(work),
                       FAIL_SEND=str(fail_send))
            for limit in range(1, len(batches)+1) if restart else [len(batches)]:
                env['STOP_AFTER'] = str(limit)
                process = subprocess.Popen(["bash", str(SCRIPT)], env=env, text=True,
                                           stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
                try:
                    _, stderr = process.communicate(timeout=10)
                except subprocess.TimeoutExpired:
                    os.killpg(process.pid, signal.SIGKILL)
                    process.communicate()
                    self.fail("consumer did not complete mock batches")
                self.assertEqual(process.returncode, -signal.SIGTERM, stderr)
            def lines(name):
                path = work / name
                return path.read_text().splitlines() if path.exists() else []
            return lines("sent"), lines("deleted"), lines("slack")

    def event(self):
        event = json.loads((ROOT / "scripts/test/sample-message.json").read_text())
        event["eventId"] = "stable-event-id"
        return event

    def test_bad_messages_are_retained_and_alert_slack(self):
        for body in ["{invalid", '{"version":1,"type":"unknown","games":[]}',
                     '{"version":1,"type":"volley.matches.new","games":[{}]}']:
            with self.subTest(body=body):
                sent, deleted, slack = self.run_consumer([body])
                self.assertEqual(sent, [])
                self.assertEqual(deleted, [])
                self.assertEqual(len(slack), 1)
                self.assertIn("Invalid notification", json.loads(slack[0])["text"])

    def test_redelivery_skips_successful_games_after_partial_failure(self):
        body = json.dumps(self.event())
        sent, deleted, slack = self.run_consumer([body, body], fail_send=2, restart=True)
        self.assertEqual(len(sent), 3)  # First game succeeds once, second game is retried.
        self.assertEqual(sent[1], sent[2])
        self.assertEqual(deleted, ["receipt-1"])
        self.assertEqual(len(slack), 1)

    def test_same_event_in_different_sqs_messages_is_not_sent_twice(self):
        body = json.dumps(self.event())
        sent, deleted, slack = self.run_consumer([body, body])
        self.assertEqual(len(sent), 2)
        self.assertEqual(deleted, ["receipt-0", "receipt-1"])
        self.assertEqual(slack, [])


if __name__ == "__main__":
    unittest.main()
