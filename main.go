package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	message       = "user SWL1234-0 pass -1 filter t/w"
	StopCharacter = "\r\n\r\n"
)

// echo "user SWL1234-0 pass -1 filter t/w" | nc rotate.aprs2.net 14580

func SocketClient(host string, port int) {

	addr := strings.Join([]string{host, strconv.Itoa(port)}, ":")
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		log.Fatalln(err)
		os.Exit(1)
	}

	defer conn.Close()

	conn.Write([]byte(message))
	conn.Write([]byte(StopCharacter))
	log.Printf("Send: %s", message)

	for {
		message, _ := bufio.NewReader(conn).ReadString('\n')
		parseMessage(message)
	}
}

func main() {

	var (
		host = "rotate.aprs2.net"
		port = 14580
	)

	SocketClient(host, port)

}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func fieldGet(data, delm string, width int) string {
	t := strings.Index(data, delm)
	if t+2+width <= len(data) && t > 0 {
		return string(data[t+1 : t+1+width])
	} else {
		return ".."
	}
}

func c2f(c string) string {
	cs, _ := strconv.ParseInt(c, 10, 64)
	return fmt.Sprintf("%3d", (cs*9/5 + 32))
}

func parseMessage(raw string) {

	//           1         2         3         4         5         6         7         8
	// 012345678901234567890123456789012345678901234567890123456789012345678901234567890
	// @210529z2856.36N/09631.40W_008/012g017t062r000p000P000b10195h67L000eMB51^M
	//                           _CSE/SPDgXXXtXXXrXXXpXXXPXXXhXXbXXXXX%type
	/*
		Where: CSE/SPD is wind direction and sustained 1 minute speed
		t is in degrees F
		r is Rain per last 60 minutes
		p is precipitation per last 24 hours (sliding 24 hour window)
		P is precip per last 24 hours since midnigh
		b is Baro in tenths of a mb
		h is humidity in percent. 00=100
		g is Gust (peak winds in last 5 minutes)
		# is the raw rain counter for remote WX stations
		See notes on remotes below
		% shows software type d=Dos, m=Mac, w=Win, etc
		type shows type of WX instrument
	*/

	var (
		ident, locat, data string

		id   string
		time string
		lat  string
		lon  string
		temp string
		rh   string
	)

	if strings.Count(raw, ":") > 0 && strings.Contains(raw, "_") {
		s1 := strings.Index(raw, ":")
		s2 := strings.Index(raw, "_")
		if s1 < s2 {
			ident = string(raw[:s1])
			locat = strings.TrimPrefix(raw[s1+1:s2], "/")
			data = strings.TrimSuffix(raw[s2:], "\n")
		} else {
			fmt.Println("DEBUG:", s1, s2, raw)
		}

		//fmt.Printf("%d:%s --- %d:%s\n", len(locat), locat, len(data), data)

		locatlen := len(locat)
		switch locatlen {
		case 26: // the standad. Why cant they all be this?
			time = locat[1:6]
			lat = locat[8:16]
			lon = locat[17:26]
		case 19: // time not included, just lat/lon
			time = "-----"
			lat = locat[1:9]
			lon = locat[10:19]
		case 25: // 25byte missing leading character
			time = locat[0:5]
			lat = locat[7:15]
			lon = locat[16:25]
		case 36, 37: //weird string inserted, time starts after *
			s1 = strings.Index(locat, "*")
			if s1 > 0 && s1 < locatlen-25 {
				time = locat[s1+1 : s1+6]
				lat = locat[s1+8 : s1+16]
				lon = locat[s1+17 : s1+26]
			}
		case 0, 7, 10, 17, 27:
			// known empty/short records
			return
		default:
			fmt.Println("LOCAT LENGTH", locatlen, "--> ", locat)
			return
		}
		// in theory if we made it this far we have a valid time/location
		// fmt.Printf("time:%sZ lat:%s lon:%s :-: %s\n", time, lat, lon, data)

		id = strings.Split(ident, ">")[0] // snag the station id for later

		datalen := len(data)
		if datalen > 3 { // nice standard fommat
			temp = fieldGet(data, "t", 3)
			rh = fieldGet(data, "h", 2)
		}
		if !isNumeric(temp) && datalen > 3 { // well drats, wasn't standard afterall. Let's try something.
			//_.../...t49h64b9908Arduino Meteo [T:9.80*C H:64.00% P:743.12mmHg] {570/600s}
			temp = fieldGet(data, "t", 2)
			rh = fieldGet(data, "h", 2)
		}
		if !isNumeric(temp) { // ugh, again? Let's try another odd format
			//_phg2280 HS7AJ 145.625MHz   T=38° H=39%  P=1008 PM 1:[15] , 2.5:[23] , 10:[26] (μg/m3)
			s1 = strings.Index(data, "T=")
			s2 = strings.Index(data, "H=")
			if (s1 != 0) && (s2 != 0) && s1 < s2 && datalen > s2+5 {
				if isNumeric(data[s1+2 : s1+4]) { //ooh, we did it maybe?
					// temp = c2f(string(data[s1+2 : s1+4])) // convert c to f (not sure if needed)
					temp = (data[s1+2 : s1+4])
					rh = data[s2+2 : s2+4]
				}
			}
		}
		if isNumeric(temp) {
			// success!
			fmt.Printf("%s %s %s %s %s %s\n", time, lat, lon, temp, rh, id)

		} else if datalen > 2 {
			// fail :-(  Bad record

			//fmt.Printf("%s %s %s %s %s %s: %s\n", time, lat, lon, temp, rh, id, strings.TrimSuffix(data, "\n"))
		}

	}

}
