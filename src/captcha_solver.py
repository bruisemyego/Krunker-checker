import hashlib,base64,json,msgpack,websocket,time

class CaptchaSolver:
 def captcha(s, challenge, salt, algorithm='SHA-256', max_iter=300000):
  for num in range(max_iter + 1):
   if algorithm == 'SHA-256':
    hash_input = f"{salt}{num}".encode('utf-8')
    hash_result = hashlib.sha256(hash_input).hexdigest()
   else:
    hash_input = f"{salt}{num}".encode('utf-8')
    hash_result = hashlib.md5(hash_input).hexdigest()
   if hash_result == challenge:
    return num
  return None

 def handle_captcha(s, ws, captcha_data, username):
  challenge = captcha_data.get('challenge')
  salt = captcha_data.get('salt')
  algorithm = captcha_data.get('algorithm', 'SHA-256')
  max_number = captcha_data.get('maxnumber', 300000)
  signature = captcha_data.get('signature')
  
  solution = s.captcha(challenge, salt, algorithm, max_number)
  
  if solution is not None:
   captcha_response = {
    'algorithm': algorithm,
    'challenge': challenge,
    'number': solution,
    'salt': salt,
    'signature': signature
   }
   b64_response = base64.b64encode(json.dumps(captcha_response).encode()).decode()
   captcha_req = ["cptR", b64_response]
   packed_captcha = msgpack.packb(captcha_req)
   ws.send(packed_captcha + b'\x00\x00', opcode=websocket.ABNF.OPCODE_BINARY)
   time.sleep(2)
   return True
  else:
   return False
