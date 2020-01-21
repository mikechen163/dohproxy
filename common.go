package main

import (

   //"fmt"
   // "io/ioutil"
    "log"
    "strings"
    "os"
    "bufio"
    "io"
  

    "bytes"

)

func is_blocked(url string, m map[string]int ) bool{
  for key, _ := range m {
     //fmt.Println("Key:", key, "Value:", value , "url:" , url)
     //fmt.Printf("%v %v %v\n",key, m[key], value)
         if strings.Contains(url,key){

            return true
         }
     }

  return false
}

func is_chn_domain(nurl string, m map[string]int ) bool{

  url := strings.TrimSpace(nurl) 
  
  if 1 == m[format_domain_name(url)]{
    return true
  }

  if strings.HasSuffix(url,".cn") || strings.HasSuffix(url,".qq.com") || strings.HasSuffix(url,".baidu.com")  {
    return true
  }

  if strings.Contains(url,"apple.com") ||  strings.Contains(url,"mzstatic.com") {
    return true
  }

  return false
}


func format_domain_name(s string) string{
    str := strings.Trim(s, " ")
    
    count2 := strings.Count(str,".")

   if count2 <= 1 {
    return str
   }
   
     if count2 == 2 {
        if strings.HasPrefix(str,"www") || strings.HasPrefix(str,"blog") || strings.HasSuffix(str,"com") || strings.HasSuffix(str,"net")  {
            nstr := strings.Split(str,".")
            return nstr[1]+"."+nstr[2]
        }
     }

     if strings.HasPrefix(str,"www") || strings.HasPrefix(str,"blog.") || strings.HasSuffix(str,".com") || strings.HasSuffix(str,".net")   {
             nstr := strings.Split(str,".")
             return strings.Join(nstr[1:],",")

      }
      return str
}



func get_config(fname string, cn_domain bool) map[string]int{
    f, err := os.Open(fname)
    if err != nil {
        log.Printf("ERROR: open %s fail:%v\n", "cn.txt")
        return nil
    }
    defer f.Close()

    r := bufio.NewReader(f)

    var m map[string]int
    m = make(map[string]int)

    //var buffer bytes.Buffer

    for {
         str , e := r.ReadString(' ')
        if e == io.EOF {
            break
        }

        if e != nil {
            log.Printf("ERROR: read  fail:%v\n", e)
            return nil
        }
         
         if (cn_domain == true){
            ns := format_domain_name(str)     
        
            m[ns] = 1
         } else {
            m[strings.TrimSpace(str)] = 1
         }
        //log.Printf("%s\n", domain)
    } //end for
    return m

}

func get_url(localBuf []byte) string {

 
    len2 := len(localBuf)
    if len2 == 0 {
        return ""
    }


    var s bytes.Buffer
    
    i := 0
    for {

        

        if  (i > (len2 -1)) {
            return s.String()
        }

        c := localBuf[i]


        if  (c == 0) {
            return s.String()
        }

        printable := false
        if (c >= 'a') && (c <= 'z') {
            printable = true
        }
        if (c >= 'A') && (c <= 'Z') {
            printable = true
        }
        if (c >= '1') && (c <= '9') {
            printable = true
        }

        if (c == '-') || (c == '_') || (c == '.') {
            printable = true
        }

        tc := string(c)
        if printable == false {
            tc = "."
            //if c != "." { //other char is invalid
                //return s.String()
            //}
        }

        //s = s + tc
        s.WriteString(tc)
        i = i + 1

    }

    return s.String()

}