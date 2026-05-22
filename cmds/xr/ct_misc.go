package main

import ()

func TestSniff(td *TD) {
	reg := td.GetRegistry()
	td.Log("Server URL: %s", reg.GetServerURL())

	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "", nil)
	td.HTTPStatusMustEqual(res, 200, "GET /")
	td.HTTPBodyMustJSON(res, "GET /")
}

func TestTD(td *TD) {
	td.Run(TestFunc1)
	td.DependsOn(TestFunc1a)
	td.DependsOn(TestFunc2)
	td.DependsOn(TestFunc2a)
	td.DependsOn(TestFunc3)
	td.DependsOn(TestFunc4)
	td.Run(TestFunc5)
	td.DependsOn(TestFunc6)
	td.DependsOn(TestFunc6a)
	td.Run(TestFunc7)
}

func TestSniffTD(td *TD) {
	td.Pass("TD Sniffer worked")
}

func TestFunc0(td *TD) {
	td.DependsOn(TestSniff)
	td.Log("testreg0 log msg")
}

func TestFunc1(td *TD) {
	td.Log("tr1 info line")
	td.Warn("just a warning - 1")
}

func TestFunc1a(td *TD) {
	/*
		td.Log("1omething happened1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")
		td.Log("2omething happened1234567890123 456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")
		td.Log("3omething happened12345678901234567890123456789012345678901234567890123456789012345 67890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")
		td.Log("4omething happened12345678901234567890123456789012345678901234567890123456789012345 67890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")
		td.Log("5omethinghappened12345678901234567890123456789012345678901234567890123456 78901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")
		td.Log("6omethinghappened1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890")
	*/
	td.Log("checking valid names")
}

func TestFunc2(td *TD) {
	td.Pass()
}

func TestFunc2a(td *TD) {
	td.Log("something happened")
	td.Warn("A warning")
	td.Pass()
}

func TestFunc3(td *TD) {
	td.Pass("subtest1")
}

func TestFunc4(td *TD) {
	td.Pass("subtest2")
	td.Skip("Not implemented")
	td.Pass()
}

func TestFunc5(td *TD) {
	td.Log("tr5 info line")
	td.Pass("subtest3")
	td.Fail("Not good")
}

func TestFunc6(td *TD) {
	td.Fail("subtest4 asd a sda sd asd asd ads a ds asd asd a ds asd asd asd a sda d asd as da ds asd ")
}

func TestFunc6a(td *TD) {
	td.Fail("subtest4 asd a sda sd asd asd ads a ds asd asd a ds asd asd asd a sda d asd as da ds asd ")
	td.Fail("xxx")
}

func TestFunc7(td *TD) {
	td.Log("something happened")
	td.Log("a b c d e f g h i j k l m n o p q r s t u v w x y z 1 2 3 4 5 6 7 8 9 0 q w e r t y u u i o p a s d f g h j k l x c v b n m q w e r t y u i o p a s d f g h j j k l")
	td.Fail("subtest5")
	// td.Fail("subtest5 asd a sda sd asd asd ads a ds asd asd a ds asd asd asd a sda d asd as da ds asd ")
	td.Pass()
}

func TestAll2(td *TD) {
	td.Run(TestGroups)
}

func TestGroups(td *TD) {
	td.Should(false, "A should fail test")
	td.Should(true, "A should pass test")
	td.ShouldEqual(1, 2, "A shouldEqual fail test")
	td.ShouldEqual(1, 1, "A shouldEqual pass test")
	// td.MustEqual(1, 2, "A Equal fail test")
}
