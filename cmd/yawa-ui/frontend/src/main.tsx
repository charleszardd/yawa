import React, {useEffect, useMemo, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {AlertCircle, CheckCircle2, Command, FolderOpen, Play, Plus, Power, RefreshCw, Save, Search, Square, Trash2, X, Zap} from 'lucide-react';
import './style.css';
import './executable.css';
import './agent.css';

type ActionType='exec'|'pick_folder'|'pick_file'|'open_path'|'url';
type CommandDefinition={name:string;description:string;type:ActionType;executable:string;args:string[];working_dir:string;path:string;url:string;hotkey:string;start_dir:string};
type SnapshotResponse={configPath:string;commands:CommandDefinition[]|null};
type Snapshot={configPath:string;commands:CommandDefinition[]};
type ApplicationOption={name:string;executable:string};
type ExecutableStatus={found:boolean;resolvedPath:string;message:string};
type AgentStatus={available:boolean;running:boolean;active:boolean;startWithWindows:boolean;executable:string;message:string};
declare global { interface Window { go:{ui:{Service:{Load:()=>Promise<SnapshotResponse>;Applications:()=>Promise<ApplicationOption[]|null>;ResolveExecutable:(value:string)=>Promise<ExecutableStatus>;BrowseExecutable:()=>Promise<string>;AgentStatus:()=>Promise<AgentStatus>;StartAgent:()=>Promise<void>;ReloadAgent:()=>Promise<void>;StopAgent:()=>Promise<void>;SetStartWithWindows:(enabled:boolean)=>Promise<void>;Save:(command:CommandDefinition,originalName:string)=>Promise<void>;Delete:(name:string)=>Promise<void>;Run:(name:string)=>Promise<void>}}}} }

const emptyCommand=():CommandDefinition=>({name:'',description:'',type:'exec',executable:'',args:[],working_dir:'',path:'',url:'',hotkey:'',start_dir:''});
const service=()=>window.go.ui.Service;

function App(){
  const [snapshot,setSnapshot]=useState<Snapshot>({configPath:'',commands:[]});
  const [selected,setSelected]=useState(''); const [draft,setDraft]=useState(emptyCommand());
  const [query,setQuery]=useState(''); const [busy,setBusy]=useState(false); const [notice,setNotice]=useState(''); const [error,setError]=useState('');
  const [applications,setApplications]=useState<ApplicationOption[]>([]); const [executableStatus,setExecutableStatus]=useState<ExecutableStatus|null>(null);
  const [agentStatus,setAgentStatus]=useState<AgentStatus>({available:false,running:false,active:false,startWithWindows:false,executable:'',message:'Checking background agent…'}); const [agentBusy,setAgentBusy]=useState(false);
  const filtered=useMemo(()=>snapshot.commands.filter(c=>(c.name+' '+c.description+' '+c.hotkey).toLowerCase().includes(query.toLowerCase())),[snapshot,query]);
  const load=async()=>{try{const response=await service().Load();setSnapshot({...response,commands:response.commands??[]})}catch(e){setError(String(e))}};
  const refreshAgentStatus=async()=>{try{const status=await service().AgentStatus();setAgentStatus(status);return status}catch(e){setError(String(e));return null}};
  useEffect(()=>{load();refreshAgentStatus();service().Applications().then(value=>setApplications(value??[])).catch(e=>setError(String(e)));const timer=window.setInterval(refreshAgentStatus,2500);return()=>window.clearInterval(timer)},[]);
  const edit=(command:CommandDefinition)=>{setSelected(command.name);setDraft({...emptyCommand(),...command,args:command.args||[]});setError('');setNotice('')};
  const create=()=>{setSelected('');setDraft(emptyCommand());setError('');setNotice('')};
  const save=async()=>{setBusy(true);setError('');try{await service().Save(draft,selected);await load();setSelected(draft.name);if(agentStatus.running){await service().ReloadAgent();setNotice('Command saved. Background shortcuts reloaded.')}else{setNotice('Command saved. Start the background agent to enable its shortcut.')}}catch(e){setError(String(e))}finally{setBusy(false);await refreshAgentStatus()}};
  const remove=async()=>{if(!selected||!confirm(`Delete “${selected}”?`))return;setBusy(true);try{await service().Delete(selected);create();await load();setNotice('Command deleted.')}catch(e){setError(String(e))}finally{setBusy(false)}};
  const run=async()=>{setBusy(true);setError('');try{await service().Run(draft.name);setNotice('Command launched.')}catch(e){setError(String(e))}finally{setBusy(false)}};
  const set=(key:keyof CommandDefinition,value:string|string[])=>setDraft(current=>({...current,[key]:value}));
  const executableAction=['exec','pick_folder','pick_file'].includes(draft.type);
  useEffect(()=>{if(!executableAction){setExecutableStatus(null);return}const timer=window.setTimeout(()=>{service().ResolveExecutable(draft.executable).then(setExecutableStatus).catch(e=>setExecutableStatus({found:false,resolvedPath:'',message:String(e)}))},250);return()=>window.clearTimeout(timer)},[draft.executable,draft.type,executableAction]);
  const browseExecutable=async()=>{try{const path=await service().BrowseExecutable();if(path)set('executable',path)}catch(e){setError(String(e))}};
  const runAgentAction=async(action:'start'|'reload'|'stop')=>{setAgentBusy(true);setError('');try{if(action==='start')await service().StartAgent();else if(action==='reload')await service().ReloadAgent();else await service().StopAgent();const status=await refreshAgentStatus();setNotice(action==='stop'?'Background agent stopped.':status?.active?'Background shortcuts are active.':status?.message??'Background agent updated.')}catch(e){setError(String(e))}finally{setAgentBusy(false);await refreshAgentStatus()}};
  const toggleStartWithWindows=async(enabled:boolean)=>{setAgentBusy(true);setError('');try{await service().SetStartWithWindows(enabled);setNotice(enabled?'Yawa will start when you sign in to Windows.':'Automatic startup disabled.')}catch(e){setError(String(e))}finally{setAgentBusy(false);await refreshAgentStatus()}};
  return <div className="shell">
    <aside>
      <div className="brand"><span className="brandmark"><Zap size={19}/></span><div><strong>yawa</strong><small>shortcut control</small></div></div>
      <button className="new" onClick={create}><Plus size={17}/> New command</button>
      <div className="search"><Search size={16}/><input value={query} onChange={e=>setQuery(e.target.value)} placeholder="Search commands"/></div>
      <div className="command-list">{filtered.map(c=><button key={c.name} className={selected===c.name?'active':''} onClick={()=>edit(c)}><span className="command-icon"><Command size={16}/></span><span><strong>{c.name}</strong><small>{c.description||c.type}</small></span>{c.hotkey&&<kbd>{c.hotkey}</kbd>}</button>)}{!filtered.length&&<p className="empty">No commands found.</p>}</div>
      <div className="agent-card"><div className="agent-heading"><span className={agentStatus.active?'status-dot running':agentStatus.running?'status-dot warning':'status-dot'}/><div><strong>{agentStatus.active?'Shortcuts active':agentStatus.running?'Agent needs attention':'Agent stopped'}</strong><small>{agentStatus.message}</small></div></div><div className="agent-actions"><button disabled={agentBusy||!agentStatus.available} onClick={()=>runAgentAction(agentStatus.running?'reload':'start')}>{agentStatus.running?<RefreshCw size={14}/>:<Power size={14}/>} {agentStatus.running?'Reload':'Start'}</button><button className="stop-agent" title="Stop agent" disabled={agentBusy||!agentStatus.running} onClick={()=>runAgentAction('stop')}><Square size={13}/></button></div><label className="startup-toggle"><input type="checkbox" checked={agentStatus.startWithWindows} disabled={agentBusy||!agentStatus.available} onChange={event=>toggleStartWithWindows(event.target.checked)}/><span>Start with Windows</span></label></div>
      <div className="config-path"><small>CONFIGURATION</small><span title={snapshot.configPath}>{snapshot.configPath||'Loading…'}</span></div>
    </aside>
    <main>
      <header><div><p>COMMAND EDITOR</p><h1>{selected?`Edit ${selected}`:'Create a command'}</h1><span>Configure an action and assign an optional global shortcut.</span></div><div className="header-actions">{selected&&<button className="danger" onClick={remove} disabled={busy}><Trash2 size={17}/></button>}<button className="secondary" onClick={run} disabled={busy||!selected}><Play size={16}/> Test</button><button className="primary" onClick={save} disabled={busy}><Save size={16}/>{busy?'Saving…':'Save command'}</button></div></header>
      {(notice||error)&&<div className={error?'banner error':'banner'}>{error||notice}<button onClick={()=>{setError('');setNotice('')}}><X size={16}/></button></div>}
      <section className="panel">
        <div className="grid two"><Field label="Name" hint="Lowercase letters, numbers, and hyphens"><input value={draft.name} onChange={e=>set('name',e.target.value)} placeholder="open-code"/></Field><Field label="Shortcut" hint="Example: Ctrl+Alt+1"><input value={draft.hotkey} onChange={e=>set('hotkey',e.target.value)} placeholder="Ctrl+Alt+1"/></Field></div>
        <Field label="Description"><input value={draft.description} onChange={e=>set('description',e.target.value)} placeholder="Choose a repository and open it in VS Code"/></Field>
        <Field label="Action type"><div className="types">{(['exec','pick_folder','pick_file','open_path','url'] as ActionType[]).map(type=><button className={draft.type===type?'selected':''} onClick={()=>set('type',type)} key={type}>{type.replace('_',' ')}</button>)}</div></Field>
        {executableAction&&<><Field label="Application" hint="Choose a detected application or browse for an .exe"><div className="executable-picker"><select value={applications.some(application=>application.executable===draft.executable)?draft.executable:''} onChange={e=>e.target.value&&set('executable',e.target.value)}><option value="">Select an installed application…</option>{applications.map(application=><option key={application.name} value={application.executable}>{application.name}</option>)}</select><button type="button" className="browse" onClick={browseExecutable}><FolderOpen size={16}/> Browse…</button></div><input value={draft.executable} onChange={e=>set('executable',e.target.value)} placeholder="Application name or C:\\path\\to\\app.exe"/><div className={executableStatus?.found?'field-status valid':'field-status'}>{executableStatus?.found?<CheckCircle2 size={14}/>:<AlertCircle size={14}/>}<span>{executableStatus?.message}{executableStatus?.found&&executableStatus.resolvedPath!==draft.executable?` ${executableStatus.resolvedPath}`:''}</span></div></Field><Field label="Arguments" hint={draft.type==='exec'?'One argument per line. Optional.':'One argument per line. Include {selected_path}.'}><textarea rows={4} value={(draft.args||[]).join('\n')} onChange={e=>set('args',e.target.value.split('\n').filter(Boolean))} placeholder={draft.type==='exec'?'--new-window\nC:\\Projects\\my-project':'--new-window\n{selected_path}'}/></Field><div className="grid two"><Field label="Working directory"><input value={draft.working_dir} onChange={e=>set('working_dir',e.target.value)} placeholder="Optional"/></Field>{draft.type!=='exec'&&<Field label="Picker start directory"><input value={draft.start_dir} onChange={e=>set('start_dir',e.target.value)} placeholder="C:\\Projects"/></Field>}</div></>}
        {draft.type==='open_path'&&<Field label="File or folder path"><input value={draft.path} onChange={e=>set('path',e.target.value)} placeholder="%USERPROFILE%\\Downloads"/></Field>}
        {draft.type==='url'&&<Field label="HTTPS URL"><input value={draft.url} onChange={e=>set('url',e.target.value)} placeholder="https://chatgpt.com"/></Field>}
      </section>
      <footer><span><Zap size={14}/> {agentStatus.active?'Global shortcuts are active.':'Start or reload the background agent to enable global shortcuts.'}</span><span>{snapshot.commands.length} command{snapshot.commands.length===1?'':'s'}</span></footer>
    </main>
  </div>
}
function Field({label,hint,children}:{label:string;hint?:string;children:React.ReactNode}){return <label className="field"><span>{label}{hint&&<small>{hint}</small>}</span>{children}</label>}
createRoot(document.getElementById('root')!).render(<App/>);
