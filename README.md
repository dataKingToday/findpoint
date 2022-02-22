# findpoint
Calculate distance between a point and a line in latitude, longitude (with Harversine Formular)

 - 프로그램명 : findpoint (https://github.com/dataKingToday/findpoint)

 - 수행작업 : 여러 개의 링크를 로딩하여 주어진 좌표로 가장 가까운 링크 위 좌표를 찾아서 
              주어진 좌표와 찾은 좌표까지 거리미터, 찾은 좌표의 경위도를 출력
    ※ 좌표P(target)와 직선AB가 아닌, 선분AB와의 수직(경위도상)인 좌표H를 구하고,
       좌표H가 좌표A,B 사이에 있는 경우, 좌표 H를
       좌표H가 좌표A,B 바깥에 있는 경우, 좌표A,B중 가까운 좌표를 출력함
       (1개의 큰링크에 여러개의 작은링크가 있는 경우에도 그 중 가장 가까운 좌표를 출력함)

 - 입력예시 : go run findpoint.go  ./findlink -links links.geojson -target 127.027268062,37.499212063 
    ※ links.geojson : 로딩할 입력파일명 (findpoint.go와 동일 폴더내에 위치해야함)
       target : “경도, 위도” 형식으로 입력
