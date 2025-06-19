from musicai_sdk import MusicAiClient
import dotenv
import os
import sys
import requests
dotenv.load_dotenv()

client = MusicAiClient(api_key=os.getenv("MUSICAI_API_KEY"))

def gen_backing_track(song_path: str):
    try:
        print(f"Iniciando processamento de: {song_path}", file=sys.stderr)
        
        # Fazer upload do arquivo de música
        print("Fazendo upload do arquivo...", file=sys.stderr)
        song_url = client.upload_file(song_path)
        print(f"Upload concluído. URL: {song_url}", file=sys.stderr)

        # Definir parâmetros do workflow para criar backing track instrumental
        workflow_params = {'inputUrl': song_url}

        # Criar o job usando o slug correto do workflow
        print("Criando job...", file=sys.stderr)
        job_response = client.add_job(
            job_name="Backing Track",
            workflow_slug='create-instrumental-backing-track',
            params=workflow_params
        )
        job_id = job_response["id"]
        print(f"Job criado com ID: {job_id}", file=sys.stderr)

        # Aguardar a conclusão do job
        print("Aguardando conclusão do job...", file=sys.stderr)
        job = client.wait_for_job_completion(job_id)
        print(f"Job finalizado com status: {job['status']}", file=sys.stderr)

        if job["status"] == "SUCCEEDED":
            print("Job bem-sucedido! Processando resultados...", file=sys.stderr)
            
            # Processar os resultados diretamente das URLs
            if "result" in job and job["result"]:
                print("Resultados encontrados no job:", job["result"], file=sys.stderr)
                
                # Procurar pela backing track (pode ter nomes diferentes)
                backing_track_url = job['result']['backing_track']
                
                return backing_track_url
                
            else:
                print("Nenhum resultado encontrado no job!", file=sys.stderr)
                print("Dados do job:", job, file=sys.stderr)
                client.delete_job(job_id)
                return None
        else:
            print(f"Job falhou com status: {job['status']}", file=sys.stderr)
            if 'error' in job:
                print(f"Erro: {job['error']}", file=sys.stderr)
            client.delete_job(job_id)
            return None
            
    except Exception as e:
        print(f"Erro durante o processamento: {str(e)}", file=sys.stderr)
        return None

if __name__ == "__main__":
    result = gen_backing_track("/root/any-song/backend/utils/audios/Seu Pereira e Coletivo 401 - Até Ontem.mp3")
    if result:
        print(f"Backing track criado com sucesso: {result}", file=sys.stderr)
    else:
        print("Falha ao criar backing track", file=sys.stderr)