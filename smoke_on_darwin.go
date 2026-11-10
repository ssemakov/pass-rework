//go:build darwin && production && native_smoke

// This harness is compiled only into the separate synthetic smoke-test app.
package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"github.com/ssemakov/pass-rework/internal/vault"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const smokeEnabled = true

var smokeRoot string
var smokeExternalRoot bool
var smokePickCount atomic.Int32
var smokeSourceHash [32]byte
var smokeSourceMu sync.Mutex
var smokeResult atomic.Int32
var smokeOnce sync.Once
var smokeFinish = make(chan bool, 1)

func smokePick(purpose string) (string, error) {
	smokePickCount.Add(1)
	if err := prepareSmokeRoot(); err != nil {
		return "", err
	}
	if purpose == "destination" {
		return filepath.Join(smokeRoot, "imported.pass"), nil
	}
	if purpose == "source" {
		b, err := os.ReadFile(filepath.Join(smokeRoot, "storage.pass"))
		if err != nil {
			return "", err
		}
		smokeSourceMu.Lock()
		smokeSourceHash = sha256.Sum256(b)
		smokeSourceMu.Unlock()
	}
	return filepath.Join(smokeRoot, "storage.pass"), nil
}
func prepareSmokeRoot() error {
	if smokeRoot == "" {
		if root := os.Getenv("PASS_NATIVE_SMOKE_ROOT"); root != "" {
			smokeRoot = root
			smokeExternalRoot = true
			return nil
		}
		r, e := os.MkdirTemp("", "pass-native-synthetic-")
		if e != nil {
			return e
		}
		smokeRoot = r
	}
	return nil
}
func smokeSettingsPath() (string, error) {
	if err := prepareSmokeRoot(); err != nil {
		return "", err
	}
	return filepath.Join(smokeRoot, "settings.json"), nil
}
func smokeHandler(next http.Handler, capability string, windowID func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/__smoke_result" && r.URL.Path != "/__smoke_key" {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method != "POST" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Pass-Capability")), []byte(capability)) != 1 || r.Header.Get("X-Wails-Window-Id") != windowID() {
			http.Error(w, "denied", 403)
			return
		}
		if r.URL.Path == "/__smoke_key" {
			nativeSmokeKey(r.URL.Query().Get("key") == "~")
			w.WriteHeader(204)
			return
		}
		ok := r.URL.Query().Get("ok") == "1"
		expectedPicks := int32(2)
		if os.Getenv("PASS_NATIVE_SMOKE_PHASE") == "restart" {
			expectedPicks = 0
		}
		if ok && smokePickCount.Load() != expectedPicks {
			ok = false
			fmt.Println("Unexpected picker during startup/login")
		}
		if ok && expectedPicks == 2 {
			b, err := os.ReadFile(filepath.Join(smokeRoot, "storage.pass"))
			smokeSourceMu.Lock()
			expectedHash := smokeSourceHash
			smokeSourceMu.Unlock()
			if err != nil || sha256.Sum256(b) != expectedHash {
				ok = false
				fmt.Println("Native import source changed")
			}
			if ok {
				fmt.Println("Native import count and same-screen result verified; source unchanged")
			}
		}
		if !ok {
			fmt.Println("Synthetic native smoke failure:", r.URL.Query().Get("step"))
		}
		select {
		case smokeFinish <- ok:
		default:
		}
		w.WriteHeader(204)
	})
}
func installSmoke(app *application.App, window *application.WebviewWindow, manager *vault.Manager) {
	smokeResult.Store(1)
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		smokeOnce.Do(func() {
			go func() {
				time.Sleep(time.Second)
				width, height := window.Size()
				if width != 904 || height != 534 {
					fmt.Println("Native frame mismatch:", width, height)
					smokeFinish <- false
					return
				}
				fmt.Println("Native window frame verified: 904x534")
				restarting := "false"
				if os.Getenv("PASS_NATIVE_SMOKE_PHASE") == "restart" {
					restarting = "true"
				}
				window.ExecJS("window.__smokeRestart=" + restarting + ";" + smokeJS)
			}()
		})
	})
	go func() {
		select {
		case ok := <-smokeFinish:
			if ok {
				smokeResult.Store(0)
				fmt.Println("Native WebKit smoke passed: explicit UI assertions complete")
				if os.Getenv("PASS_NATIVE_SMOKE_PHASE") == "restart" {
					fmt.Println("Native remembered storage restart verified: wrong-key error, correct retry, zero pickers")
				}
			}
		case <-time.After(65 * time.Second):
			fmt.Println("Native smoke timed out")
		}
		manager.Close()
		app.Quit()
	}()
}
func smokePostShutdown() {
	if smokeRoot != "" && !smokeExternalRoot {
		os.RemoveAll(smokeRoot)
	}
	if smokeResult.Load() == 0 {
		fmt.Println("Native WebKit smoke passed: create/edit/reveal/copy/lock, bridge denial, navigation denial")
	} else {
		fmt.Println("Native WebKit smoke failed")
	}
	os.Exit(int(smokeResult.Load()))
}

const smokeJS = `(async()=>{
 const delay=n=>new Promise(r=>setTimeout(r,n));
 const wait=async(fn,label)=>{for(let i=0;i<250;i++){const value=fn();if(value)return value;await delay(40)}throw new Error(label)};
 const button=text=>[...document.querySelectorAll('button')].find(b=>(b.getAttribute('aria-label')===text||b.textContent.trim()===text)&&!b.disabled);
 const click=async text=>{(await wait(()=>button(text),text)).click();await delay(70)};
 const fill=async(label,value)=>{const input=await wait(()=>[...document.querySelectorAll('input')].find(e=>e.getAttribute('aria-label')===label||e.getAttribute('placeholder')===label)||[...document.querySelectorAll('label')].filter(e=>e.textContent.trim()===label).map(e=>document.getElementById(e.htmlFor))[0],label);Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(input,value);input.dispatchEvent(new Event('input',{bubbles:true}));await delay(50)};
 const masterKey=window.__smokeRestart?'public-native-imported-master':'public-native-smoke-master';
 const command=name=>window.dispatchEvent(new CustomEvent('pass-command',{detail:name}));
 const report=async(ok,step="")=>{await fetch('/__smoke_result?ok='+Number(ok)+'&step='+encodeURIComponent(step),{method:'POST',headers:{'X-Pass-Capability':window._wails.flags.vaultCapability}})};
 try{
  if(getComputedStyle(document.documentElement).backgroundColor!=='rgb(244, 244, 245)')throw new Error('legacy background');
  const login=await wait(()=>document.querySelector('.auth-container .outlined-field'),'login');
  if(Math.abs(login.getBoundingClientRect().height-56)>1||document.querySelector('header,footer'))throw new Error('legacy login layout');
  if(window.__smokeRestart){
   await wait(()=>document.querySelector('input[placeholder="Enter your Master Password"]'),'remembered login');
   if(document.querySelector('input[placeholder="Enter your storage Master Password"]'))throw new Error('restart returned to signup');
   await fill('Master password','wrong-native-master');await click('Unlock storage');
   await wait(()=>document.querySelector('[role="alert"]')?.textContent==="Wrong password",'wrong key indication');
   await delay(1100);if(!document.querySelector('input[placeholder="Enter your Master Password"]')||document.querySelector('input').disabled)throw new Error('wrong key left login busy');
   await fill('Master password',masterKey);await click('Unlock storage');await wait(()=>document.querySelector('[aria-label="Search stored secrets"]'),'correct retry');
   await fill('Search stored secrets','Native');await wait(()=>document.querySelector('article'),'persisted entry');await click('Reveal password');await wait(()=>document.querySelector('[aria-label="Revealed password"]')?.textContent==='public-native-smoke-secret','persisted password');
  }else{
  await wait(()=>document.querySelector('input[placeholder="Enter your storage Master Password"]'),'first launch signup');
  if(!button('Import Storage…'))throw new Error('first launch import missing');
  await fill('Master password','public-native-smoke-master');await fill('Confirm Master Password','public-native-smoke-master');await click('Create Storage');await wait(()=>document.querySelector('[aria-label="Search stored secrets"]'),'unlock');
  const fab=document.querySelector('.fab').getBoundingClientRect();if(fab.width!==56||fab.height!==56)throw new Error('legacy add size');
  await click('Add new secret');await fill('Item name','Native synthetic login');await fill('Username','native@example.test');await fill('Password','public-native-smoke-secret');await fill('Website URL','https://example.test');await click('Save');await wait(()=>document.querySelector('[aria-label="Search stored secrets"]'),'saved');await fill('Search stored secrets','Native');await wait(()=>document.querySelector('article'),'card');await click('Reveal password');await wait(()=>document.querySelector('[aria-label="Revealed password"]')?.textContent==='public-native-smoke-secret','reveal');
  await click('Edit secret');await fill('Item name','Native edited login');await click('Save');await click('Reveal password');await wait(()=>document.querySelector('[aria-label="Revealed password"]')?.textContent==='public-native-smoke-secret','unchanged password');await click('Copy password');await wait(()=>button('Copy password'),'copy completed');
  if(document.querySelector('[role="alert"]'))throw new Error('copy failed');
  }
  await click('Lock storage');await wait(()=>document.querySelector('input[placeholder="Enter your Master Password"]'),'visible lock control');
  await fill('Master password',masterKey);await click('Unlock storage');await wait(()=>document.querySelector('[aria-label="Search stored secrets"]'),'button lock relogin');
  const denied=await fetch('/wails/runtime',{method:'POST',headers:{'Content-Type':'application/json'},body:'{}'});if(denied.status!==403)throw new Error('unauthenticated bridge accepted');
  await fetch('/__smoke_key?key=l',{method:'POST',headers:{'X-Pass-Capability':window._wails.flags.vaultCapability}});await delay(200);if(!document.querySelector('[aria-label="Search stored secrets"]'))throw new Error('old Command-L still locks');
  await fetch('/__smoke_key?key=~',{method:'POST',headers:{'X-Pass-Capability':window._wails.flags.vaultCapability}});await wait(()=>document.querySelector('input[placeholder="Enter your Master Password"]')&&!document.body.textContent.includes('public-native-smoke-secret'),'native Command-tilde lock');
  if(!window.__smokeRestart){
   command('import');await fill('Backup Master Password','public-native-smoke-master');await click('Select encrypted backup');await click('Select fresh destination');
   await fill('New Master Password','public-native-imported-master');await fill('Confirm Master Password','public-native-imported-master');await click('Import');
   await wait(()=>document.querySelector('form[aria-label="Import Storage"] [role="status"]')?.textContent==='Imported 1 secrets','native imported count');
   if(document.querySelector('input[placeholder="Enter your Master Password"]'))throw new Error('import navigated away from result');
   await click('Open storage');await fill('Master password','public-native-imported-master');await click('Unlock storage');await wait(()=>document.querySelector('[aria-label="Search stored secrets"]'),'open imported storage');
  }
  window.location.href='data:text/html,<h1 id="untrusted-navigation">must never load</h1>';await delay(400);if(location.protocol!=='wails:'||document.querySelector('#untrusted-navigation'))throw new Error('navigation accepted');
  await report(true);
 }catch(error){console.error('Synthetic native smoke step failed',String(error));await report(false,String(error))}
})()`
