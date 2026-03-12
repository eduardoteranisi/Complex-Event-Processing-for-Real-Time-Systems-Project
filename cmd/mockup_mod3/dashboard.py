import streamlit as st
import socket
import json
import threading
import pandas as pd
import pydeck as pdk
import time
from streamlit_autorefresh import st_autorefresh
from datetime import datetime

# ==========================================
# CONFIGURAÇÃO DE INFRAESTRUTURA
# ==========================================
UDP_IP = "127.0.0.1" 
UDP_PORT = 8888

CENTER_LAT = -18.91
CENTER_LON = -48.27

EVENT_COLORS = {
    "VOLTAGE_DROP": [255, 165, 0, 200],   
    "OVERVOLTAGE": [255, 0, 0, 200],      
    "UNDERVOLTAGE": [0, 0, 255, 200],     
    "CRITICAL_DROP": [0, 0, 0, 255],      
    "NOISE": [128, 0, 128, 200],          
    "PING": [0, 255, 0, 150],             
    "DEFAULT": [128, 128, 128, 150]       
}

st.set_page_config(page_title="Módulo 3 - Mockup", layout="wide")

# ==========================================
# THREAD DE ESCUTA (BLINDADA CONTRA REFRESH)
# ==========================================
@st.cache_resource
def start_udp_listener():
    """Garante que a porta 8888 só seja aberta UMA vez, independente do autorefresh."""
    events_list = []
    lock = threading.Lock()
    
    def listen():
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        
        # Tenta usar SO_REUSEPORT (Ajuda muito no Linux/Ubuntu)
        try:
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEPORT, 1)
        except AttributeError:
            pass 
            
        try:
            sock.bind((UDP_IP, UDP_PORT))
            print(f"[Painel Python] 📡 Escutando UDP Broadcast em {UDP_IP}:{UDP_PORT}")
        except OSError:
            print("[Painel Python] Thread secundária ignorada. A principal já está escutando.")
            return # Se a porta já estiver em uso pela nossa própria thread, apenas sai silenciosamente

        while True:
            try:
                data, addr = sock.recvfrom(2048)
                
                # Descomente a linha abaixo SE quiser ver absolutamente tudo que chega no terminal
                # print(f"Recebi pacote de {addr}: {data.decode('utf-8')}") 
                
                event_data = json.loads(data.decode('utf-8'))
                
                new_event = {
                    "timestamp": datetime.now().strftime("%H:%M:%S"),
                    "lat": event_data['local'][0],
                    "lon": event_data['local'][1],
                    "type": event_data['critical_event_type'], 
                    "size": 1.5
                }
                
                new_event['color'] = EVENT_COLORS.get(new_event['type'], EVENT_COLORS["DEFAULT"])

                with lock:
                    events_list.append(new_event)
                    if len(events_list) > 100:
                        events_list.pop(0)

            except KeyError as e:
                print(f"[Erro de Formato JSON] O Go enviou um pacote, mas faltou o campo: {e}")
                print(f"Pacote recebido: {data.decode('utf-8')}")
            except Exception as e:
                print(f"[Erro Geral] {e}")
    # Inicia a thread em background
    t = threading.Thread(target=listen, daemon=True)
    t.start()
    
    return lock, events_list

# Puxa a lista e o cadeado da memória cache (nunca recria)
lock, raw_events = start_udp_listener()

# ==========================================
# INTERFACE GRÁFICA (UI)
# ==========================================
# Atualiza a tela a cada 1 segundo (1000 milissegundos)
st_autorefresh(interval=1000, key="datarefresh")

st.title("⚡ Módulo 3 - Painel Georreferenciado de Anomalias Elétricas")

with lock:
    df = pd.DataFrame(raw_events)

col_mapa, col_log = st.columns([3, 1])

with col_mapa:
    view_state = pdk.ViewState(
        latitude=CENTER_LAT,
        longitude=CENTER_LON,
        zoom=11.5,
        pitch=40, 
    )

    if not df.empty:
        layer = pdk.Layer(
            "ScatterplotLayer",
            df,
            pickable=True,
            opacity=0.8,
            stroked=True,
            filled=True,
            radius_scale=6,
            radius_min_pixels=5,
            radius_max_pixels=25,
            get_position="[lon, lat]",
            get_fill_color="color",
            get_line_color=[0, 0, 0, 255], 
            get_radius="size * 100", 
        )
        tooltip = {
            "html": "<b>Tipo:</b> {type}<br/><b>Hora:</b> {timestamp}",
            "style": {"backgroundColor": "steelblue", "color": "white"}
        }
    else:
        layer = pdk.Layer("ScatterplotLayer", pd.DataFrame())
        tooltip = None

    st.pydeck_chart(pdk.Deck(
        layers=[layer],
        initial_view_state=view_state,
        map_provider="carto", # Usa o provedor gratuito que não exige API Key
        map_style="light",    # Estilo claro da Carto
        tooltip=tooltip
    ))

with col_log:
    st.subheader("Log de Alertas")
    if st.button("Limpar Mapa"):
        with lock:
            raw_events.clear()
        st.rerun()

    if not df.empty:
        df_log = df.iloc[::-1].head(15)[['timestamp', 'type']]
        def color_map(val):
            return f'color: rgb({EVENT_COLORS.get(val, EVENT_COLORS["DEFAULT"])[0]},{EVENT_COLORS.get(val, EVENT_COLORS["DEFAULT"])[1]},{EVENT_COLORS.get(val, EVENT_COLORS["DEFAULT"])[2]})'
            
        st.dataframe(df_log.style.applymap(color_map, subset=['type']), hide_index=True, use_container_width=True)
    else:
        st.write("Aguardando anomalias na rede...")
