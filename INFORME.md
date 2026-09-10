# Informe TP0
Hernán Lagarde, 105827
## Introducción
Hago un recorrido por los ejercicios, las dificultades encontradas, los detalles más importantes, etc.

## Docker Compose
Es cuestión de hacer copypaste del cliente 1 y cambiarle el nombre de la definición, el containername y el AGENCY_ID.
Sin embargo, es más prolijo reutilizar confs anteriores, voy a implementar eso dado que es mejor

Al correr el ejercicio veo que los clientes fallan en recibir respuesta y deben intentar varias veces,
dado que el servidor les responde uno a uno de forma no concurrente, entonces deben intentar varias veces

## Docker Network
En Server, pongo
```
ports:
  - 5678:5678
```


## Docker Volumes
En el compose. en la parte del cliente
```
volumes:
  - ./input:/input
  - ./output:/output
```
y en el dockerfile de cliente hago copy a la carpeta input

## Short read/write
En GO, io.EOF puede ser un error o un indicador del final de la lectura, debo propagarlo o no en función de si leí todo
lo que tenía que leer o no
Para el write, itero indefinidamente hasta escribir todo o recibir error, en ese caso lo propago

En Python, recibir b'' en el read, significa que la conexión se cerró.
Para el write, el error que se recibe es socket module error

## Protocolo de comunicación
A partir de una línea del csv, cuento bytes totales y le agrego un magic, el tipo, el agencyid y el tamaño
para el mensaje bet, le agrego un payload
```
[magic] 4 B
[tipo] 1 B
[agencyid] 1 B
[payload len] 4 B
[payload] variable
```
al final el magic es inutil e innecesario dado que no implementé resincronización. confiamos que TCP envíá todo 
ordenado y correctamente
Hay 3 tipos de mensajes: BET (un solo bet), BAT(muchos bets separados por \n), STOP(el cliente dandole consentimiento al
servidor para arrancar la votación)

## Batch
Implementado el mensaje BAT, se le agrega al cliente un delay de 1ms entre batch y batch para no saturar el socket y que
pase la prueba de memoria

## Multithreading
En el sevidor se implementó 1 hilo principal, 1 hilo para detectar el shutdown y tirar abajo la barrera, y n hilos segun
cuantas conexiones haya

## Graceful shutdown
Para acelerar el shutdown lo más posible, se llevó hasta los receivers y senders implementados. También se implementó un
hilo que detecta el shutdown para tirar abajo la barrera del server. Dado que Golang no acepta sobrecarga, tuve que
implementar funciones con distinto nombre, unas con el graceful shutdown y otras sin, para que el test de
short read/write(que requiere una firma específica) pueda andar sin problemas. Sin embargo, ambas comparten la mísma
lógica en lo que respecta a short read/write