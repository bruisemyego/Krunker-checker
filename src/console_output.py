import os,time,threading
from colorama import Fore,Style,init
init(autoreset=True)

class C2:
 def __init__(s):s.st=time.time();s.ta=0;s.tca=0;s.tkr=0;s.tfa=0;s.lk=threading.Lock();s.lut=0;s.ui=2.0;s.ic={"1k+":0,"5k+":0,"10k+":0,"50k+":0,"100k+":0};s.lc={"10+":0,"30+":0,"50+":0,"80+":0,"100+":0}
 def cs(s):os.system('cls'if os.name=='nt'else'clear')
 def dis(s):ct=time.time();return(ct-s.lut>=s.ui)and(setattr(s,'lut',ct)or True)if ct-s.lut>=s.ui else False
 def c1(s,tc):s.ui=3.7 if tc>=500 else 3.0 if tc>=200 else 2.0 if tc>=100 else 1.0
 def iv1(s,iv):
  with s.lk:
   if iv>=100000:s.ic["100k+"]+=1
   elif iv>=50000:s.ic["50k+"]+=1
   elif iv>=10000:s.ic["10k+"]+=1
   elif iv>=5000:s.ic["5k+"]+=1
   elif iv>=1000:s.ic["1k+"]+=1
 def iv2(s,lv):
  with s.lk:
   if lv>=100:s.lc["100+"]+=1
   elif lv>=80:s.lc["80+"]+=1
   elif lv>=50:s.lc["50+"]+=1
   elif lv>=30:s.lc["30+"]+=1
   elif lv>=10:s.lc["10+"]+=1
 def cpm(s):et=time.time()-s.st;return round((s.tca/et)*60,1)if et>0 else 0
 def eta(s):
  if s.tca==0:return"Calculating..."
  et=time.time()-s.st;al=s.ta-s.tca
  if al<=0:return"Complete!"
  rt=s.tca/et if et>0 else 0
  if rt==0:return"Calculating..."
  es=al/rt
  if es<60:return f"{int(es)}s"
  elif es<3600:return f"{int(es//60)}m {int(es%60)}s"
  else:h=int(es//3600);m=int((es%3600)//60);return f"{h}h {m}m"
 def ps(s,fu=False):
  if not fu and not s.dis():return
  with s.lk:al=s.ta-s.tca;pp=round((s.tca/s.ta)*100)if s.ta>0 else 0;s.cs()
  print(f"{Fore.WHITE}Progress: {Fore.YELLOW}{s.tca}{Fore.WHITE}/{Fore.YELLOW}{s.ta}{Fore.WHITE} ({Fore.YELLOW}{pp}{Fore.WHITE}%)")
  print(f"{Fore.WHITE}Accounts remaining: {Fore.YELLOW}{al}")
  print(f"{Fore.GREEN}Total KR found: {Fore.YELLOW}{s.tkr:,}")
  print(f"{Fore.GREEN}Hits: {Fore.YELLOW}{s.tfa}")
  print()
  print(f"{Fore.CYAN}Inventory:")
  print(f"{Fore.CYAN}  {Fore.YELLOW}1K+{Fore.CYAN}   inv: {Fore.YELLOW}{s.ic['1k+']}")
  print(f"{Fore.CYAN}  {Fore.YELLOW}5K+{Fore.CYAN}   inv: {Fore.YELLOW}{s.ic['5k+']}")
  print(f"{Fore.YELLOW}  {Fore.YELLOW}10K+{Fore.YELLOW}  inv: {Fore.YELLOW}{s.ic['10k+']}")
  print(f"{Fore.YELLOW}  {Fore.YELLOW}50K+{Fore.YELLOW}  inv: {Fore.YELLOW}{s.ic['50k+']}")
  print(f"{Fore.RED}  {Fore.YELLOW}100K+{Fore.RED} inv: {Fore.YELLOW}{s.ic['100k+']}")
  print()
  print(f"{Fore.MAGENTA}Level:")
  print(f"{Fore.CYAN}  LVL {Fore.YELLOW}10+{Fore.CYAN} : {Fore.YELLOW}{s.lc['10+']}")
  print(f"{Fore.CYAN}  LVL {Fore.YELLOW}30+{Fore.CYAN} : {Fore.YELLOW}{s.lc['30+']}")
  print(f"{Fore.YELLOW}  LVL {Fore.YELLOW}50+{Fore.YELLOW} : {Fore.YELLOW}{s.lc['50+']}")
  print(f"{Fore.YELLOW}  LVL {Fore.YELLOW}80+{Fore.YELLOW} : {Fore.YELLOW}{s.lc['80+']}")
  print(f"{Fore.RED}  LVL {Fore.YELLOW}100+{Fore.RED}: {Fore.YELLOW}{s.lc['100+']}")
  print()
  print(f"{Fore.MAGENTA}Speed: {Fore.YELLOW}{s.cpm()}{Fore.MAGENTA} accounts/min")
  print(f"{Fore.MAGENTA}ETA: {Fore.YELLOW}{s.eta()}")
  print()

console=C2()
