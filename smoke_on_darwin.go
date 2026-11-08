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
		if r.URL.Path != "/__smoke_result" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != "POST" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Pass-Capability")), []byte(capability)) != 1 || r.Header.Get("X-Wails-Window-Id") != windowID() {
			http.Error(w, "denied", 403)
			return
		}
		ok := r.URL.Query().Get("ok") == "1"
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
		smokeOnce.Do(func() { go func() { time.Sleep(time.Second); window.ExecJS(smokeJS) }() })
	})
	go func() {
		select {
		case ok := <-smokeFinish:
			if ok {
				smokeResult.Store(0)
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
 const button=(text)=>[...document.querySelectorAll('button')].find(b=>{const c=b.cloneNode(true);c.querySelectorAll('[aria-hidden]').forEach(n=>n.remove());return c.textContent.trim().startsWith(text)&&!b.disabled});
 const click=async text=>{(await wait(()=>button(text),text)).click();await delay(70)};
 const fill=async(label,value)=>{const input=await wait(()=>[...document.querySelectorAll('label')].find(e=>e.textContent.trim()===label)?.querySelector('input'),label);Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(input,value);input.dispatchEvent(new Event('input',{bubbles:true}));await delay(50)};
 const report=async ok=>{await fetch('/__smoke_result?ok='+Number(ok),{method:'POST',headers:{'X-Pass-Capability':window._wails.flags.vaultCapability}})};
 try{
  await click('Create a vault');await click('Choose vault location');await fill('Master password','public-native-smoke-master');await fill('Confirm master password','public-native-smoke-master');await click('Create vault');await wait(()=>button('Lock vault'),'unlock');
  await click('+ New entry');await fill('Entry name','Native synthetic login');await fill('Username','native@example.test');await fill('Password','public-native-smoke-secret');await click('Save entry');await wait(()=>button('Native synthetic login'),'entry saved');await click('Native synthetic login');await click('Reveal password');await wait(()=>document.querySelector('[aria-label="Revealed password"]')?.textContent==='public-native-smoke-secret','reveal');
  await click('Edit entry');await fill('Entry name','Native edited login');await click('Save entry');await click('Reveal password');await wait(()=>document.querySelector('[aria-label="Revealed password"]')?.textContent==='public-native-smoke-secret','unchanged password');await click('Copy password');await wait(()=>document.body.textContent.includes('Copied.'),'clipboard');
  const denied=await fetch('/wails/runtime',{method:'POST',headers:{'Content-Type':'application/json'},body:'{}'});if(denied.status!==403)throw new Error('unauthenticated bridge accepted');
  await click('Lock vault');await wait(()=>document.body.textContent.includes('Your secrets,')&&!document.body.textContent.includes('public-native-smoke-secret'),'lock');
  window.location.href='data:text/html,<h1 id="untrusted-navigation">must never load</h1>';await delay(400);if(location.protocol!=='wails:'||document.querySelector('#untrusted-navigation'))throw new Error('navigation accepted');
  await report(true);
 }catch(error){console.error('Synthetic native smoke step failed',String(error));await report(false)}
})()`
