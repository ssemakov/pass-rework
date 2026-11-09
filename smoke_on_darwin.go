//go:build darwin && production && native_smoke

// This harness is compiled only into the separate synthetic smoke-test app.
package main

import (
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
var smokeResult atomic.Int32
var smokeOnce sync.Once
var smokeFinish = make(chan bool, 1)

func smokePick(purpose string) (string, error) {
	if smokeRoot == "" {
		r, e := os.MkdirTemp("", "pass-native-synthetic-")
		if e != nil {
			return "", e
		}
		smokeRoot = r
	}
	return filepath.Join(smokeRoot, "vault.pass"), nil
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
				window.ExecJS(smokeJS)
			}()
		})
	})
	go func() {
		select {
		case ok := <-smokeFinish:
			if ok {
				smokeResult.Store(0)
				fmt.Println("Native WebKit smoke passed: explicit UI assertions complete")
			}
		case <-time.After(65 * time.Second):
			fmt.Println("Native smoke timed out")
		}
		manager.Close()
		app.Quit()
	}()
}
func smokePostShutdown() {
	if smokeRoot != "" {
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
 const command=name=>window.dispatchEvent(new CustomEvent('pass-command',{detail:name}));
 const report=async(ok,step="")=>{await fetch('/__smoke_result?ok='+Number(ok)+'&step='+encodeURIComponent(step),{method:'POST',headers:{'X-Pass-Capability':window._wails.flags.vaultCapability}})};
 try{
  if(getComputedStyle(document.documentElement).backgroundColor!=='rgb(244, 244, 245)')throw new Error('legacy background');
  const login=await wait(()=>document.querySelector('.auth-container .outlined-field'),'login');
  if(Math.abs(login.getBoundingClientRect().height-56)>1||document.querySelector('header,footer'))throw new Error('legacy login layout');
  command('create');await click('Select storage path');await fill('Master password','public-native-smoke-master');await click('Continue');await fill('Confirm Master Password','public-native-smoke-master');await click('Create Storage');await wait(()=>document.querySelector('[aria-label="Search stored secrets"]'),'unlock');
  const fab=document.querySelector('.fab').getBoundingClientRect();if(fab.width!==56||fab.height!==56)throw new Error('legacy add size');
  await click('Add new secret');await fill('Item name','Native synthetic login');await fill('Username','native@example.test');await fill('Password','public-native-smoke-secret');await fill('Website URL','https://example.test');await click('Save');await wait(()=>document.querySelector('[aria-label="Search stored secrets"]'),'saved');await fill('Search stored secrets','Native');await wait(()=>document.querySelector('article'),'card');await click('Reveal password');await wait(()=>document.querySelector('[aria-label="Revealed password"]')?.textContent==='public-native-smoke-secret','reveal');
  await click('Edit secret');await fill('Item name','Native edited login');await click('Save');await click('Reveal password');await wait(()=>document.querySelector('[aria-label="Revealed password"]')?.textContent==='public-native-smoke-secret','unchanged password');await click('Copy password');await wait(()=>button('Copy password'),'copy completed');
  if(document.querySelector('[role="alert"]'))throw new Error('copy failed');
  const denied=await fetch('/wails/runtime',{method:'POST',headers:{'Content-Type':'application/json'},body:'{}'});if(denied.status!==403)throw new Error('unauthenticated bridge accepted');
  await fetch('/__smoke_key?key=l',{method:'POST',headers:{'X-Pass-Capability':window._wails.flags.vaultCapability}});await delay(200);if(!document.querySelector('[aria-label="Search stored secrets"]'))throw new Error('old Command-L still locks');
  await fetch('/__smoke_key?key=~',{method:'POST',headers:{'X-Pass-Capability':window._wails.flags.vaultCapability}});await wait(()=>document.querySelector('input[placeholder="Enter your Master Password"]')&&!document.body.textContent.includes('public-native-smoke-secret'),'native Command-tilde lock');
  window.location.href='data:text/html,<h1 id="untrusted-navigation">must never load</h1>';await delay(400);if(location.protocol!=='wails:'||document.querySelector('#untrusted-navigation'))throw new Error('navigation accepted');
  await report(true);
 }catch(error){console.error('Synthetic native smoke step failed',String(error));await report(false,String(error))}
})()`
