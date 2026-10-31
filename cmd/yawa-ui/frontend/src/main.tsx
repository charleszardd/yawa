import React, {useEffect, useMemo, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {Command, Play, Plus, Save, Search, Trash2, X, Zap} from 'lucide-react';
import './style.css';

type ActionType='exec'|'pick_folder'|'pick_file'|'open_path'|'url';
type CommandDefinition={name:string;description:string;type:ActionType;executable:string;args:string[];working_dir:string;path:string;url:string;hotkey:string;start_dir:string};
type Snapshot={configPath:string;commands:CommandDefinition[]};
declare global { interface Window { go:{ui:{Service:{Load:()=>Promise<Snapshot>;Save:(command:CommandDefinition,originalName:string)=>Promise<void>;Delete:(name:string)=>Promise<void>;Run:(name:string)=>Promise<void>}}}} }

const emptyCommand=():CommandDefinition=>({name:'',description:'',type:'exec',executable:'',args:[],working_dir:'',path:'',url:'',hotkey:'',start_dir:''});
const service=()=>window.go.ui.Service;

function App(){
  const [snapshot,setSnapshot]=useState<Snapshot>({configPath:'',commands:[]});
  const [selected,setSelected]=useState(''); const [draft,setDraft]=useState(emptyCommand());
  const [query,setQuery]=useState(''); const [busy,setBusy]=useState(false); const [notice,setNotice]=useState(''); const [error,setError]=useState('');
  const filtered=useMemo(()=>snapshot.commands.filter(c=>(c.name+' '+c.description+' '+c.hotkey).toLowerCase().includes(query.toLowerCase())),[snapshot,query]);
  const load=async()=>{try{setSnapshot(await service().Load())}catch(e){setError(String(e))}};
  useEffect(()=>{load()},[]);
  const edit=(command:CommandDefinition)=>{setSelected(command.name);setDraft({...emptyCommand(),...command,args:command.args||[]});setError('');setNotice('')};
  const create=()=>{setSelected('');setDraft(emptyCommand());setError('');setNotice('')};
  const save=async()=>{setBusy(true);setError('');try{await service().Save(draft,selected);await load();setSelected(draft.name);setNotice('Command saved. Restart the agent to apply hotkey changes.')}catch(e){setError(String(e))}finally{setBusy(false)}};
  const remove=async()=>{if(!selected||!confirm(`Delete “${selected}”?`))return;setBusy(true);try{await service().Delete(selected);create();await load();setNotice('Command deleted.')}catch(e){setError(String(e))}finally{setBusy(false)}};
  const run=async()=>{setBusy(true);setError('');try{await service().Run(draft.name);setNotice('Command launched.')}catch(e){setError(String(e))}finally{setBusy(false)}};
  const set=(key:keyof CommandDefinition,value:string|string[])=>setDraft(current=>({...current,[key]:value}));
  const executableAction=['exec','pick_folder','pick_file'].includes(draft.type);
  return <div className="shell">
    <aside>
      <div className="brand"><span className="brandmark"><Zap size={19}/></span><div><strong>yawa</strong><small>shortcut control</small></div></div>
      <button className="new" onClick={create}><Plus size={17}/> New command</button>
      <div className="search"><Search size={16}/><input value={query} onChange={e=>setQuery(e.target.value)} placeholder="Search commands"/></div>
      <div className="command-list">{filtered.map(c=><button key={c.name} className={selected===c.name?'active':''} onClick={()=>edit(c)}><span className="command-icon"><Command size={16}/></span><span><strong>{c.name}</strong><small>{c.description||c.type}</small></span>{c.hotkey&&<kbd>{c.hotkey}</kbd>}</button>)}{!filtered.length&&<p className="empty">No commands found.</p>}</div>
      <div className="config-path"><small>CONFIGURATION</small><span title={snapshot.configPath}>{snapshot.configPath||'Loading…'}</span></div>
    </aside>
    <main>
      <header><div><p>COMMAND EDITOR</p><h1>{selected?`Edit ${selected}`:'Create a command'}</h1><span>Configure an action and assign an optional global shortcut.</span></div><div className="header-actions">{selected&&<button className="danger" onClick={remove} disabled={busy}><Trash2 size={17}/></button>}<button className="secondary" onClick={run} disabled={busy||!selected}><Play size={16}/> Test</button><button className="primary" onClick={save} disabled={busy}><Save size={16}/>{busy?'Saving…':'Save command'}</button></div></header>
      {(notice||error)&&<div className={error?'banner error':'banner'}>{error||notice}<button onClick={()=>{setError('');setNotice('')}}><X size={16}/></button></div>}
      <section className="panel">
        <div className="grid two"><Field label="Name" hint="Lowercase letters, numbers, and hyphens"><input value={draft.name} onChange={e=>set('name',e.target.value)} placeholder="open-code"/></Field><Field label="Shortcut" hint="Example: Ctrl+Alt+1"><input value={draft.hotkey} onChange={e=>set('hotkey',e.target.value)} placeholder="Ctrl+Alt+1"/></Field></div>
        <Field label="Description"><input value={draft.description} onChange={e=>set('description',e.target.value)} placeholder="Choose a repository and open it in VS Code"/></Field>
        <Field label="Action type"><div className="types">{(['exec','pick_folder','pick_file','open_path','url'] as ActionType[]).map(type=><button className={draft.type===type?'selected':''} onClick={()=>set('type',type)} key={type}>{type.replace('_',' ')}</button>)}</div></Field>
        {executableAction&&<><Field label="Executable" hint="Use an application on PATH or its full .exe path"><input value={draft.executable} onChange={e=>set('executable',e.target.value)} placeholder="code"/></Field><Field label="Arguments" hint="One argument per line. Picker actions require {selected_path}."><textarea rows={4} value={(draft.args||[]).join('\n')} onChange={e=>set('args',e.target.value.split('\n').filter(Boolean))} placeholder={'--new-window\n{selected_path}'}/></Field><div className="grid two"><Field label="Working directory"><input value={draft.working_dir} onChange={e=>set('working_dir',e.target.value)} placeholder="Optional"/></Field>{draft.type!=='exec'&&<Field label="Picker start directory"><input value={draft.start_dir} onChange={e=>set('start_dir',e.target.value)} placeholder="C:\\Projects"/></Field>}</div></>}
        {draft.type==='open_path'&&<Field label="File or folder path"><input value={draft.path} onChange={e=>set('path',e.target.value)} placeholder="%USERPROFILE%\\Downloads"/></Field>}
        {draft.type==='url'&&<Field label="HTTPS URL"><input value={draft.url} onChange={e=>set('url',e.target.value)} placeholder="https://chatgpt.com"/></Field>}
      </section>
      <footer><span><Zap size={14}/> Hotkeys are active while <code>yawa agent</code> is running.</span><span>{snapshot.commands.length} command{snapshot.commands.length===1?'':'s'}</span></footer>
    </main>
  </div>
}
function Field({label,hint,children}:{label:string;hint?:string;children:React.ReactNode}){return <label className="field"><span>{label}{hint&&<small>{hint}</small>}</span>{children}</label>}
createRoot(document.getElementById('root')!).render(<App/>);
