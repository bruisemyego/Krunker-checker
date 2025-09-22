import websocket,msgpack,ssl,time,random
from captcha_solver import CaptchaSolver

ws_url = "wss://social.krunker.io/ws"

class WSHandler:
 def __init__(s):s.captcha=CaptchaSolver()
 def ua(s):return f"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{random.choice(['119.0.0.0','118.0.0.0','117.0.0.0','116.0.0.0'])} Safari/537.36"

 def fetch_profile_ws(s, username, token=None, proxy=None):
  try:
   ws = websocket.WebSocket(sslopt={"cert_reqs": ssl.CERT_NONE})
   headers = {
    'User-Agent': s.ua(),
    'Origin': 'https://krunker.io'
   }
   
   if proxy:
    proxy_parts = proxy.replace('http://','').replace('https://','')
    if '@' in proxy_parts:
     auth_part, host_part = proxy_parts.split('@')
     username_proxy, password_proxy = auth_part.split(':')
     host, port = host_part.split(':')
     ws.connect(ws_url, header=headers, http_proxy_host=host, http_proxy_port=int(port),
               http_proxy_auth=(username_proxy, password_proxy), timeout=15)
    else:
     host, port = proxy_parts.split(':')
     ws.connect(ws_url, header=headers, http_proxy_host=host, http_proxy_port=int(port), timeout=15)
   else:
    ws.connect(ws_url, header=headers, timeout=15)
   
   if token:
    login_req = ["_0", 0, "login", token]
    packed_req = msgpack.packb(login_req)
    ws.send(packed_req + b'\x00\x00', opcode=websocket.ABNF.OPCODE_BINARY)
    
    login_success = False
    for _ in range(10):
     try:
      ws.settimeout(3)
      data = ws.recv()
      if len(data) >= 2:
       data = data[:-2]
      response = msgpack.unpackb(data, raw=False)
      
      if isinstance(response, list) and len(response) >= 1:
       if response[0] == "cpt":
        if not s.captcha.handle_captcha(ws, response[1], username):
         ws.close()
         return None
        continue
       elif response[0] == "a" and len(response) >= 4:
        a1 = response[3]
        login_success = True
        username = a1
        break
     except:
      continue
    
    if not login_success:
     ws.close()
     return None
   
   profile_req = ["r", "profile", username]
   packed_req = msgpack.packb(profile_req)
   ws.send(packed_req + b'\x00\x00', opcode=websocket.ABNF.OPCODE_BINARY)
   
   for _ in range(15):
    try:
     ws.settimeout(5)
     data = ws.recv()
     if len(data) >= 2:
      data = data[:-2]
     response = msgpack.unpackb(data, raw=False)
     
     if isinstance(response, list) and len(response) >= 1:
      if response[0] == "cpt":
       if not s.captcha.handle_captcha(ws, response[1], username):
        ws.close()
        return None
       profile_req = ["r", "profile", username]
       packed_req = msgpack.packb(profile_req)
       ws.send(packed_req + b'\x00\x00', opcode=websocket.ABNF.OPCODE_BINARY)
       continue
      elif len(response) >= 4 and response[3] is not None:
       if isinstance(response[3], dict):
        player_data = response[3]
        player_name = player_data.get('player_name', '')
        if player_name:
         stats = {
          'player_name': player_name,
          'player_id': player_data.get('player_id', 0),
          'player_score': player_data.get('player_score', 0),
          'player_funds': player_data.get('player_funds', 0),
          'player_skinvalue': player_data.get('player_skinvalue', 0)
         }
         ws.close()
         return stats
    except:
     continue
   
   ws.close()
   return None
  except Exception as e:
   print(f"WebSocket error for {username}: {str(e)}")
   return None