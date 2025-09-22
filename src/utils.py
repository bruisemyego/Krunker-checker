import random

def LA(count=None):
 try:
  accounts=[line.strip()for line in open("data/accounts.txt","r",encoding="utf-8",errors="ignore")if line.strip()and":"in line]
  if count and count>0:
   return random.sample(accounts,min(count,len(accounts)))
  return accounts
 except:print("data/accounts.txt not found");return[]

def LP():
 try:return[line.strip()for line in open("data/proxies.txt","r",encoding="utf-8",errors="ignore")if line.strip()]
 except:print("data/proxies.txt not found");return[]