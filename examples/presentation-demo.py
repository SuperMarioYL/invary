import subprocess
for name,expected in [('sample_deepseek_output.json',0),('sample_glm_output.json',1)]:
 p=subprocess.run(['go','run','.','check','--trace','examples/'+name,'--schema','examples/tool-schema.json'],capture_output=True,text=True)
 print(p.stdout,end='');print('command exit:',p.returncode)
 if p.returncode!=expected:raise RuntimeError(p.stderr)
