import {describe,it,expect} from 'vitest';
import {SessionFence} from './fence';
import {waitForCapability} from './api';
describe('session delivery',()=>{
 it('rejects a promise/status captured before lock or a new authentication',()=>{const f=new SessionFence();const pending=f.capture();expect(f.accepts(pending)).toBe(true);f.invalidate();expect(f.accepts(pending)).toBe(false);const authenticated=f.capture();f.invalidate();expect(f.accepts(authenticated)).toBe(false)});
 it('waits for native flags injected after runtime import',async()=>{let flag:unknown;setTimeout(()=>{flag='synthetic-capability'},25);expect(await waitForCapability(()=>flag,500)).toBe('synthetic-capability')});
 it('fails closed if native configuration never arrives',async()=>{await expect(waitForCapability(()=>undefined,20)).rejects.toThrow('authorization')});
});
