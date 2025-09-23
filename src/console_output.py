import os
import time
import threading
from colorama import Fore, Style, init

init(autoreset=True)

class C2:
    def __init__(self):
        self.start_time = time.time()
        self.total_accounts = 0
        self.total_checked_accounts = 0
        self.total_kr = 0
        self.total_found_accounts = 0
        self.lock = threading.Lock()
        self.last_update_time = 0
        self.update_interval = 2.0
        
        self.inventory_counts = {
            "1k+": 0,
            "5k+": 0,
            "10k+": 0,
            "50k+": 0,
            "100k+": 0
        }
        
        self.level_counts = {
            "10+": 0,
            "30+": 0,
            "50+": 0,
            "80+": 0,
            "100+": 0
        }
    
    def clear_screen(self):
        os.system('cls' if os.name == 'nt' else 'clear')
    
    def dis(self):
        current_time = time.time()
        if current_time - self.last_update_time >= self.update_interval:
            self.last_update_time = current_time
            return True
        return False
    
    def c1(self, thread_count):
        if thread_count >= 500:
            self.update_interval = 3.7  #update every 3.7 seconds for very high thread counts
        elif thread_count >= 200:
            self.update_interval = 3.0  #update every 3 seconds for high thread counts
        elif thread_count >= 100:
            self.update_interval = 2.0  #update every 2 seconds for medium thread counts
        else:
            self.update_interval = 1.0  #update every 1 second for low thread counts
    
    def iv1(self, inv_value):
        with self.lock:
            if inv_value >= 100000:
                self.inventory_counts["100k+"] += 1
            elif inv_value >= 50000:
                self.inventory_counts["50k+"] += 1
            elif inv_value >= 10000:
                self.inventory_counts["10k+"] += 1
            elif inv_value >= 5000:
                self.inventory_counts["5k+"] += 1
            elif inv_value >= 1000:
                self.inventory_counts["1k+"] += 1
    
    def iv2(self, level):
        with self.lock:
            if level >= 100:
                self.level_counts["100+"] += 1
            elif level >= 80:
                self.level_counts["80+"] += 1
            elif level >= 50:
                self.level_counts["50+"] += 1
            elif level >= 30:
                self.level_counts["30+"] += 1
            elif level >= 10:
                self.level_counts["10+"] += 1
    
    def calculate_cpm(self):
        elapsed_time = time.time() - self.start_time
        if elapsed_time > 0:
            return round((self.total_checked_accounts / elapsed_time) * 60, 1)
        return 0
    
    def calculate_eta(self):
        if self.total_checked_accounts == 0:
            return "Calculating..."
        
        elapsed_time = time.time() - self.start_time
        accounts_left = self.total_accounts - self.total_checked_accounts
        
        if accounts_left <= 0:
            return "Complete!"
        
        rate = self.total_checked_accounts / elapsed_time if elapsed_time > 0 else 0
        if rate == 0:
            return "Calculating..."
        
        eta_seconds = accounts_left / rate
        
        if eta_seconds < 60:
            return f"{int(eta_seconds)}s"
        elif eta_seconds < 3600:
            return f"{int(eta_seconds // 60)}m {int(eta_seconds % 60)}s"
        else:
            hours = int(eta_seconds // 3600)
            minutes = int((eta_seconds % 3600) // 60)
            return f"{hours}h {minutes}m"
    
    def print_stats(self, force_update=False):
        if not force_update and not self.dis():
            return
            
        with self.lock:
            accounts_left = self.total_accounts - self.total_checked_accounts
            progress_percent = round((self.total_checked_accounts / self.total_accounts) * 100) if self.total_accounts > 0 else 0
            
            self.clear_screen()
            
        print(f"{Fore.WHITE}Progress: {Fore.YELLOW}{self.total_checked_accounts}{Fore.WHITE}/{Fore.YELLOW}{self.total_accounts}{Fore.WHITE} ({Fore.YELLOW}{progress_percent}{Fore.WHITE}%)")
        print(f"{Fore.WHITE}Accounts remaining: {Fore.YELLOW}{accounts_left}")
        print(f"{Fore.GREEN}Total KR found: {Fore.YELLOW}{self.total_kr:,}")
        print(f"{Fore.GREEN}Accounts saved: {Fore.YELLOW}{self.total_found_accounts}")
        print()
        
        print(f"{Fore.CYAN}Inventory:")
        print(f"{Fore.CYAN}  {Fore.YELLOW}1K+{Fore.CYAN}   inv: {Fore.YELLOW}{self.inventory_counts['1k+']}")
        print(f"{Fore.CYAN}  {Fore.YELLOW}5K+{Fore.CYAN}   inv: {Fore.YELLOW}{self.inventory_counts['5k+']}")
        print(f"{Fore.YELLOW}  {Fore.YELLOW}10K+{Fore.YELLOW}  inv: {Fore.YELLOW}{self.inventory_counts['10k+']}")
        print(f"{Fore.YELLOW}  {Fore.YELLOW}50K+{Fore.YELLOW}  inv: {Fore.YELLOW}{self.inventory_counts['50k+']}")
        print(f"{Fore.RED}  {Fore.YELLOW}100K+{Fore.RED} inv: {Fore.YELLOW}{self.inventory_counts['100k+']}")
        print()
        
        print(f"{Fore.MAGENTA}Level:")
        print(f"{Fore.CYAN}  LVL {Fore.YELLOW}10+{Fore.CYAN} : {Fore.YELLOW}{self.level_counts['10+']}")
        print(f"{Fore.CYAN}  LVL {Fore.YELLOW}30+{Fore.CYAN} : {Fore.YELLOW}{self.level_counts['30+']}")
        print(f"{Fore.YELLOW}  LVL {Fore.YELLOW}50+{Fore.YELLOW} : {Fore.YELLOW}{self.level_counts['50+']}")
        print(f"{Fore.YELLOW}  LVL {Fore.YELLOW}80+{Fore.YELLOW} : {Fore.YELLOW}{self.level_counts['80+']}")
        print(f"{Fore.RED}  LVL {Fore.YELLOW}100+{Fore.RED}: {Fore.YELLOW}{self.level_counts['100+']}")
        print()
        
        print(f"{Fore.MAGENTA}Speed: {Fore.YELLOW}{self.calculate_cpm()}{Fore.MAGENTA} accounts/min")
        print(f"{Fore.MAGENTA}ETA: {Fore.YELLOW}{self.calculate_eta()}")
        print()

console = C2()
