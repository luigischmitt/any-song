#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import sys
import json
import shutil
from pathlib import Path
from utils.backing_track import gen_backing_track
from utils.catch_lyrics import lyrics
from utils.song_name import extract_song_info

def process_uploaded_song(audio_path: str, song_title: str = None, artist_name: str = None, output_base_dir: str = "output", upload_to_gcs: bool = False) -> dict:
    """
    Processa uma música enviada pelo usuário:
    1. Cria pasta organizada para a música
    2. Gera backing track via Music.AI
    3. Busca letras via MusixMatch
    4. Organiza tudo numa estrutura de pastas
    5. Opcionalmente, prepara para upload no GCS
    
    Args:
        audio_path (str): Caminho do arquivo de áudio enviado
        song_title (str): Título da música (opcional)
        artist_name (str): Nome do artista (opcional)
        output_base_dir (str): Diretório base para salvar resultados
        upload_to_gcs (bool): Se deve preparar para upload no GCS
    
    Returns:
        dict: Resultado do processamento com caminhos dos arquivos
    """
    
    try:
        if not os.path.exists(audio_path):
            return {"error": f"Arquivo de áudio não encontrado: {audio_path}"}
        
        # Extrair informações da música se não fornecidas
        if not song_title or not artist_name:
            extracted_info = extract_song_info(os.path.basename(audio_path))
            song_title = song_title or extracted_info.get('title', 'Música Desconhecida')
            artist_name = artist_name or extracted_info.get('artist', 'Artista Desconhecido')
        
        # Criar nome limpo para a pasta
        clean_name = f"{artist_name} - {song_title}".replace('/', '_').replace('\\', '_')
        song_folder = os.path.join(output_base_dir, clean_name)
        os.makedirs(song_folder, exist_ok=True)
        
        print(f"🎵 Processando: {clean_name}", file=sys.stderr)
        print(f"📁 Pasta de saída: {song_folder}", file=sys.stderr)
        
        # Copiar arquivo original para a pasta
        original_filename = os.path.basename(audio_path)
        original_path = os.path.join(song_folder, f"original_{original_filename}")
        shutil.copy2(audio_path, original_path)
        print(f"✅ Arquivo original copiado para: {original_path}", file=sys.stderr)
        
        results = {
            "success": True,
            "song_title": song_title,
            "artist_name": artist_name,
            "song_folder": song_folder,
            "original_file": original_path,
            "files": {},
            "gcs_folder": clean_name if upload_to_gcs else None  # Pasta no GCS
        }
        
        # Passo 1: Gerar backing track com Music.AI
        print("🎼 Gerando backing track com Music.AI...", file=sys.stderr)
        try:
            backing_track_url = gen_backing_track(audio_path)
            if backing_track_url:
                # Salvar informações do backing track
                backing_info = {
                    "backing_track_url": backing_track_url,
                    "generated_at": str(os.path.getctime(audio_path)),
                    "original_file": original_path
                }
                
                backing_info_path = os.path.join(song_folder, "backing_track_info.json")
                with open(backing_info_path, 'w', encoding='utf-8') as f:
                    json.dump(backing_info, f, indent=2, ensure_ascii=False)
                
                results["files"]["backing_track_info"] = backing_info_path
                results["backing_track_url"] = backing_track_url
                print(f"✅ Backing track gerado: {backing_track_url}", file=sys.stderr)
            else:
                print("⚠️ Não foi possível gerar backing track", file=sys.stderr)
                results["warnings"] = results.get("warnings", [])
                results["warnings"].append("Backing track não pôde ser gerado")
        
        except Exception as e:
            print(f"❌ Erro ao gerar backing track: {str(e)}", file=sys.stderr)
            results["warnings"] = results.get("warnings", [])
            results["warnings"].append(f"Erro no backing track: {str(e)}")
        
        # Passo 2: Buscar letras com MusixMatch
        print("📝 Buscando letras...", file=sys.stderr)
        try:
            lyrics_data = lyrics(song_title, artist_name)
            if lyrics_data and "lyrics" in lyrics_data:
                # Salvar letras em arquivo JSON
                lyrics_path = os.path.join(song_folder, f"{clean_name}_lyrics.json")
                with open(lyrics_path, 'w', encoding='utf-8') as f:
                    json.dump(lyrics_data, f, indent=2, ensure_ascii=False)
                
                # Salvar letras em arquivo de texto também
                lyrics_txt_path = os.path.join(song_folder, f"{clean_name}_lyrics.txt")
                with open(lyrics_txt_path, 'w', encoding='utf-8') as f:
                    f.write(lyrics_data["lyrics"])
                
                results["files"]["lyrics_json"] = lyrics_path
                results["files"]["lyrics_txt"] = lyrics_txt_path
                results["lyrics"] = lyrics_data["lyrics"]
                print(f"✅ Letras salvas em: {lyrics_path}", file=sys.stderr)
            else:
                print("⚠️ Não foi possível encontrar letras", file=sys.stderr)
                results["warnings"] = results.get("warnings", [])
                results["warnings"].append("Letras não encontradas")
        
        except Exception as e:
            print(f"❌ Erro ao buscar letras: {str(e)}", file=sys.stderr)
            results["warnings"] = results.get("warnings", [])
            results["warnings"].append(f"Erro na busca de letras: {str(e)}")
        
        # Passo 3: Criar arquivo de resumo do processamento
        summary = {
            "song_info": {
                "title": song_title,
                "artist": artist_name,
                "clean_name": clean_name
            },
            "processing_info": {
                "processed_at": str(os.path.getctime(audio_path)),
                "original_file": original_path,
                "song_folder": song_folder,
                "upload_to_gcs": upload_to_gcs
            },
            "files": results["files"],
            "backing_track_url": results.get("backing_track_url"),
            "has_lyrics": "lyrics_json" in results["files"],
            "warnings": results.get("warnings", [])
        }
        
        summary_path = os.path.join(song_folder, "processing_summary.json")
        with open(summary_path, 'w', encoding='utf-8') as f:
            json.dump(summary, f, indent=2, ensure_ascii=False)
        
        results["files"]["summary"] = summary_path
        results["processing_summary"] = summary
        
        print(f"✅ Processamento concluído!", file=sys.stderr)
        print(f"📁 Todos os arquivos salvos em: {song_folder}", file=sys.stderr)
        print(f"📋 Resumo salvo em: {summary_path}", file=sys.stderr)
        
        if upload_to_gcs:
            print(f"🔄 Preparado para upload no GCS na pasta: {clean_name}", file=sys.stderr)
        
        return results
        
    except Exception as e:
        print(f"❌ Erro no processamento: {str(e)}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)
        return {"error": f"Erro no processamento: {str(e)}"}

def main():
    """Função principal para teste"""
    if len(sys.argv) < 2:
        print("Uso: python process_song.py <caminho_do_audio> [titulo] [artista] [--gcs]", file=sys.stderr)
        sys.exit(1)
    
    audio_path = sys.argv[1]
    song_title = sys.argv[2] if len(sys.argv) > 2 and not sys.argv[2].startswith('--') else None
    artist_name = sys.argv[3] if len(sys.argv) > 3 and not sys.argv[3].startswith('--') else None
    upload_to_gcs = '--gcs' in sys.argv
    
    result = process_uploaded_song(audio_path, song_title, artist_name, upload_to_gcs=upload_to_gcs)
    
    if "error" in result:
        print(f"❌ Erro: {result['error']}", file=sys.stderr)
        sys.exit(1)
    else:
        print("✅ Música processada com sucesso!", file=sys.stderr)
        # Imprimir APENAS o JSON no stdout para o Go conseguir fazer parse
        print(json.dumps(result, indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main() 