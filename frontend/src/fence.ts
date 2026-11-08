// A fulfilled backend promise may already be queued when lock occurs.
export class SessionFence {
 private revision=0;
 invalidate(){this.revision++}
 capture(){return this.revision}
 accepts(revision:number){return revision===this.revision}
}
