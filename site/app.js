const demo = document.querySelector('#demo');
const next = document.querySelector('#demo-next');
const reset = document.querySelector('#demo-reset');
const log = document.querySelector('#demo-log');
const counter = document.querySelector('#demo-counter');
const steps = [
  {log:'Ready. Three nodes hold the same signed HearthPack.',label:'Start failure sequence'},
  {log:'Step 1 — Publisher A is offline. Replicas B and C remain available.',label:'Tamper with replica B'},
  {log:'Step 2 — Replica B changed locally. Its file hash no longer matches the signed manifest.',label:'Verify the damaged copy'},
  {log:'Step 3 — Verification failed. Replica B rejects the damaged content.',label:'Recover from replica C'},
  {log:'Step 4 — Replica C supplied a valid copy. B verified the signature and restored the pack.',label:'Sequence complete'}
];
let current = 0;
function render(){
  demo.dataset.step = String(current);
  const a = demo.querySelector('[data-node="a"]');
  const b = demo.querySelector('[data-node="b"]');
  a.className = 'demo-node publisher-node' + (current >= 1 ? ' offline' : '');
  b.className = 'demo-node' + (current === 2 || current === 3 ? ' bad' : '') + (current === 4 ? ' repairing' : '');
  a.querySelector('small').textContent = current >= 1 ? 'Offline' : 'Online';
  b.querySelector('small').textContent = current === 2 ? 'Changed' : current === 3 ? 'Rejected' : current === 4 ? 'Restored' : 'Verified';
  log.textContent = steps[current].log;
  next.textContent = steps[current].label;
  next.disabled = current === 4;
  counter.textContent = current + ' / 4';
}
next.addEventListener('click',()=>{if(current<4){current++;render();}});
reset.addEventListener('click',()=>{current=0;render();});
document.querySelector('#copy-command').addEventListener('click',async(e)=>{
  const value='cd hearthmesh/mvp\ngo test ./...\nsh demo.sh';
  try{
    await navigator.clipboard.writeText(value);
    e.currentTarget.textContent='Copied';
    setTimeout(()=>e.currentTarget.textContent='Copy commands',1600);
  }catch{
    e.currentTarget.textContent='Select and copy above';
  }
});
render();
