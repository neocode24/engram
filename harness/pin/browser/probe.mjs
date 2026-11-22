// engram serve 의 /pin 화면을 크롬으로 열어 확인한다. 대역 서버가 없다. 서버는 이미 떠 있어야 한다.
//
// 확인하는 것은 probe6(archie 설계 2.3절)과 같다.
//   탭이 넷이다(미리알림 탭이 없다)
//   `맞음`을 누르면 반영되어 제안 카드가 줄어든다
//   콘솔의 오류와 경고가 0건이다
//   127.0.0.1 밖으로 나가는 요청이 0건이다
//   서버를 내린 뒤 일정 시간 동안 페이지 이동이 0번이다(연결 실패를 로그인 만료로 읽고 다시 불러오지 않는다)
//
// 사용: node probe.mjs <서버 주소 예: http://127.0.0.1:8420> <playwright-core 경로> <크롬 실행 파일> <서버 pid> [내린 뒤 기다릴 초]
// 서버 pid 를 주면 화면을 확인한 뒤 그 프로세스를 직접 멈춘다. 멈추지 않을 거면 pid 에 0 을 준다.
import { createRequire } from 'node:module';
const [base, pw, chrome, pidArg, waitArg] = process.argv.slice(2);
const { chromium } = createRequire(import.meta.url)(pw);
const pid = Number(pidArg || 0), waitSec = Number(waitArg || 75);
const origin = new URL(base).origin;

const browser = await chromium.launch({ executablePath: chrome, headless: true });
const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 }, locale: 'ko-KR' });
let external = 0, externalUrls = [];
await ctx.route((url) => url.origin !== origin, (r) => { external++; externalUrls.push(r.request().url().slice(0, 80)); r.abort(); });
const page = await ctx.newPage();
const problems = []; let navs = 0;
page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') problems.push(m.type() + ': ' + m.text().slice(0, 140)); });
page.on('pageerror', (e) => problems.push('pageerror: ' + e.message));
page.on('framenavigated', (f) => { if (f === page.mainFrame()) navs++; });

await page.goto(`${base}/pin`);
await page.waitForSelector('#view .card');
const tabs = await page.$$eval('nav [role=tab]', (e) => e.map((x) => x.textContent.trim()));
const cardsBefore = await page.locator('#view .ask').count();
const itemsInView = await page.locator('#view').innerText();
await page.locator('#view .ask button', { hasText: '맞음' }).first().click();
await page.waitForFunction((n) => document.querySelectorAll('#view .ask').length < n, cardsBefore, { timeout: 30000 });
const cardsAfter = await page.locator('#view .ask').count();
for (const t of tabs) { await page.locator('nav [role=tab]', { hasText: t }).first().click(); await page.waitForTimeout(250); }
const text = await page.locator('#view').innerText();
const evidenceShown = /킥오프|견적서/.test(itemsInView + text);
const problemsWhileUp = problems.slice(), navsUp = navs;

let stopped = false;
if (pid > 0) { process.kill(pid, 'SIGTERM'); stopped = true; }
await page.waitForTimeout(waitSec * 1000);
const down = await page.evaluate(() => ({
  authbar_hidden: document.getElementById('authbar').hidden,
  head: document.querySelector('header').innerText.replace(/\s+/g, ' ').slice(0, 120),
  view_has_content: document.querySelectorAll('#view *').length > 0,
})).catch((e) => ({ error: String(e).slice(0, 80) }));
const result = {
  tabs, cardsBefore, cardsAfter, evidence_shown: evidenceShown,
  problems_while_up: problemsWhileUp, external_requests: external, external_urls: externalUrls,
  server_stopped: stopped, wait_seconds: waitSec, navigations_after_server_down: navs - navsUp,
  url_after_wait: page.url().replace(/:\d+/, ':PORT'), down,
};
console.log(JSON.stringify(result, null, 1));
await browser.close();
const pass = tabs.length === 4 && !tabs.includes('미리알림') && cardsAfter < cardsBefore && problemsWhileUp.length === 0 &&
  external === 0 && navs - navsUp === 0 && down.authbar_hidden === true;
console.log(pass ? 'PASS' : 'FAIL');
process.exit(pass ? 0 : 1);
